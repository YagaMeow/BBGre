package service

import (
	"os"
	"strings"
)

// ID3Tags is the set of metadata that can be written into an MP3 file. The
// fields map onto standard ID3v2.4 frames.
type ID3Tags struct {
	Title     string
	Artist    string
	Album     string
	Year      string
	Track     string
	Genre     string
	Composer  string
	BPM       string
	Mood      string
	Featuring string
	Copyright string
	Comment   string
	Lyrics    string
	Cover     []byte
	CoverMIME string
}

func synchsafeBytes(n int) []byte {
	return []byte{byte((n >> 21) & 0x7f), byte((n >> 14) & 0x7f), byte((n >> 7) & 0x7f), byte(n & 0x7f)}
}

func buildFrame(id string, data []byte) []byte {
	header := make([]byte, 10)
	copy(header[0:4], id)
	copy(header[4:8], synchsafeBytes(len(data)))
	return append(header, data...)
}

// textFrame builds a UTF-8 encoded text frame (v2.4 supports encoding byte 3).
func textFrame(id, value string) []byte {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return buildFrame(id, append([]byte{0x03}, []byte(value)...))
}

// txxxFrame builds a user defined text frame such as "Featuring".
func txxxFrame(desc, value string) []byte {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	data := []byte{0x03}
	data = append(data, []byte(desc)...)
	data = append(data, 0x00)
	data = append(data, []byte(value)...)
	return buildFrame("TXXX", data)
}

// commentFrame builds a COMM frame: encoding + language + description + text.
func commentFrame(text string) []byte {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	data := []byte{0x03}
	data = append(data, 'e', 'n', 'g')
	data = append(data, 0x00)
	data = append(data, []byte(text)...)
	return buildFrame("COMM", data)
}

// lyricsFrame builds a USLT frame: encoding + language + descriptor + lyrics.
func lyricsFrame(text string) []byte {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	data := []byte{0x03}
	data = append(data, 'e', 'n', 'g')
	data = append(data, 0x00)
	data = append(data, []byte(text)...)
	return buildFrame("USLT", data)
}

// pictureFrame builds an APIC frame with a front cover picture.
func pictureFrame(pic []byte, mime string) []byte {
	if len(pic) == 0 {
		return nil
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	data := []byte{0x00} // latin-1 encoding for mime/description
	data = append(data, []byte(mime)...)
	data = append(data, 0x00)
	data = append(data, 0x03) // picture type: front cover
	data = append(data, 0x00) // empty description terminator
	data = append(data, pic...)
	return buildFrame("APIC", data)
}

// BuildID3v24 serialises the tags into a complete ID3v2.4 tag (including the
// 10 byte header). It returns nil when there is nothing to write.
func BuildID3v24(t ID3Tags) []byte {
	var frames []byte
	appendFrame := func(f []byte) {
		if f != nil {
			frames = append(frames, f...)
		}
	}
	appendFrame(textFrame("TIT2", t.Title))
	appendFrame(textFrame("TPE1", t.Artist))
	appendFrame(textFrame("TALB", t.Album))
	appendFrame(textFrame("TDRC", t.Year))
	appendFrame(textFrame("TRCK", t.Track))
	appendFrame(textFrame("TCON", t.Genre))
	appendFrame(textFrame("TCOM", t.Composer))
	appendFrame(textFrame("TBPM", t.BPM))
	appendFrame(textFrame("TMOO", t.Mood))
	appendFrame(textFrame("TCOP", t.Copyright))
	appendFrame(txxxFrame("Featuring", t.Featuring))
	appendFrame(commentFrame(t.Comment))
	appendFrame(lyricsFrame(t.Lyrics))
	appendFrame(pictureFrame(t.Cover, t.CoverMIME))

	if len(frames) == 0 {
		return nil
	}
	header := make([]byte, 10)
	copy(header[0:3], "ID3")
	header[3] = 0x04 // version 2.4.0
	header[4] = 0x00
	header[5] = 0x00
	copy(header[6:10], synchsafeBytes(len(frames)))
	return append(header, frames...)
}

// WriteID3 replaces the ID3 tags of an MP3 in place while leaving the audio
// frames untouched. The old ID3v2 tag (if any) and a trailing ID3v1 tag are
// stripped before the freshly built tag is prepended.
func WriteID3(filePath string, t ID3Tags) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	audioStart := 0
	if len(data) >= 10 && string(data[0:3]) == "ID3" {
		size := synchsafe(data[6:10])
		audioStart = 10 + size
		if data[5]&0x10 != 0 { // footer present
			audioStart += 10
		}
		if audioStart > len(data) {
			audioStart = len(data)
		}
	}

	audioEnd := len(data)
	if audioEnd > 128 && string(data[audioEnd-128:audioEnd-125]) == "TAG" {
		audioEnd -= 128
	}
	if audioStart > audioEnd {
		audioStart = audioEnd
	}

	audio := data[audioStart:audioEnd]
	tag := BuildID3v24(t)
	out := make([]byte, 0, len(tag)+len(audio))
	out = append(out, tag...)
	out = append(out, audio...)

	tmp := filePath + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filePath)
}

// DetectImageMIME returns the MIME type based on the image magic bytes.
func DetectImageMIME(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case len(data) >= 8 && string(data[0:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a"):
		return "image/gif"
	default:
		return ""
	}
}