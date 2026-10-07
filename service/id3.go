package service

import (
	"bytes"
	"encoding/binary"
	"strings"
	"unicode/utf16"
)

// mp3Info holds the metadata extracted from an MP3 file.
type mp3Info struct {
	Title       string
	Artist      string
	Album       string
	Year        string
	Track       string
	Genre       string
	Composer    string
	BPM         string
	Mood        string
	Comment     string
	Featuring   string
	Copyright   string
	Lyrics      string
	Picture     []byte
	PictureMIME string
	Duration    float64
}

// ParseMP3 extracts ID3 metadata (title, artist, album, embedded lyrics and
// cover art) plus an estimated duration from raw MP3 bytes.
func ParseMP3(data []byte) *mp3Info {
	info := &mp3Info{}
	tagSize := 0

	if len(data) >= 10 && string(data[0:3]) == "ID3" {
		version := data[3]
		flags := data[5]
		size := synchsafe(data[6:10])
		tagSize = 10 + size
		if flags&0x10 != 0 { // footer present
			tagSize += 10
		}
		body := data[10:]
		if size > len(body) {
			size = len(body)
		}
		body = body[:size]
		if flags&0x80 != 0 { // whole-tag unsynchronisation
			body = deunsync(body)
		}
		parseID3Frames(version, body, info)
	} else {
		parseID3v1(data, info)
	}

	audio := data
	if tagSize > 0 && tagSize < len(data) {
		audio = data[tagSize:]
	}
	// Strip a trailing ID3v1 tag before estimating the duration.
	if len(audio) > 128 && string(audio[len(audio)-128:len(audio)-125]) == "TAG" {
		audio = audio[:len(audio)-128]
	}
	info.Duration = estimateDuration(audio)
	return info
}

func synchsafe(b []byte) int {
	return int(b[0]&0x7f)<<21 | int(b[1]&0x7f)<<14 | int(b[2]&0x7f)<<7 | int(b[3]&0x7f)
}

func parseID3Frames(version byte, body []byte, info *mp3Info) {
	idLen, headerLen := 4, 10
	if version == 2 {
		idLen, headerLen = 3, 6
	}

	// Skip an extended header if present (we do not need its contents).
	off := 0
	for off+headerLen <= len(body) {
		id := string(body[off : off+idLen])
		if strings.Trim(id, "\x00") == "" {
			break // reached padding
		}
		if !validFrameID(id) {
			break
		}
		var size int
		switch {
		case version == 2:
			size = int(body[off+3])<<16 | int(body[off+4])<<8 | int(body[off+5])
		case version == 4:
			size = synchsafe(body[off+4 : off+8])
		default:
			size = int(binary.BigEndian.Uint32(body[off+4 : off+8]))
		}
		start := off + headerLen
		end := start + size
		if size <= 0 || end > len(body) {
			break
		}
		applyFrame(id, body[start:end], info)
		off = end
	}
}

func validFrameID(id string) bool {
	if len(id) == 0 {
		return false
	}
	for _, c := range []byte(id) {
		if !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func applyFrame(id string, frame []byte, info *mp3Info) {
	if len(frame) == 0 {
		return
	}
	switch id {
	case "TIT2", "TT2":
		info.Title = decodeTextFrame(frame)
	case "TPE1", "TP1":
		info.Artist = decodeTextFrame(frame)
	case "TALB", "TAL":
		info.Album = decodeTextFrame(frame)
	case "TDRC", "TYER", "TYE":
		info.Year = decodeTextFrame(frame)
	case "TRCK", "TRK":
		info.Track = decodeTextFrame(frame)
	case "TCON", "TCO":
		info.Genre = decodeTextFrame(frame)
	case "TCOM", "TCM":
		info.Composer = decodeTextFrame(frame)
	case "TBPM", "TBP":
		info.BPM = decodeTextFrame(frame)
	case "TMOO":
		info.Mood = decodeTextFrame(frame)
	case "TCOP", "TCR":
		info.Copyright = decodeTextFrame(frame)
	case "COMM", "COM":
		if info.Comment == "" {
			info.Comment = parseUSLT(frame)
		}
	case "TXXX", "TXX":
		if desc, val := parseTXXX(frame); strings.EqualFold(strings.TrimSpace(desc), "Featuring") {
			info.Featuring = val
		}
	case "USLT", "ULT":
		info.Lyrics = parseUSLT(frame)
	case "APIC", "PIC":
		if mime, pic := parseAPIC(id, frame); len(pic) > 0 {
			info.Picture = pic
			info.PictureMIME = mime
		}
	}
}

func parseID3v1(data []byte, info *mp3Info) {
	if len(data) < 128 {
		return
	}
	t := data[len(data)-128:]
	if string(t[0:3]) != "TAG" {
		return
	}
	info.Title = strings.TrimRight(latin1(t[3:33]), "\x00 ")
	info.Artist = strings.TrimRight(latin1(t[33:63]), "\x00 ")
	info.Album = strings.TrimRight(latin1(t[63:93]), "\x00 ")
}

// ---- text decoding -------------------------------------------------------

func decodeTextFrame(frame []byte) string {
	if len(frame) == 0 {
		return ""
	}
	return strings.TrimRight(decodeWithEnc(frame[0], frame[1:]), "\x00")
}

func decodeWithEnc(enc byte, b []byte) string {
	switch enc {
	case 0:
		return latin1(b)
	case 1:
		return utf16Decode(b, true)
	case 2:
		return utf16Decode(b, false)
	case 3:
		return string(b)
	default:
		return latin1(b)
	}
}

func latin1(b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func utf16Decode(b []byte, allowBOM bool) string {
	order := binary.ByteOrder(binary.LittleEndian)
	if allowBOM && len(b) >= 2 {
		if b[0] == 0xFF && b[1] == 0xFE {
			b = b[2:]
		} else if b[0] == 0xFE && b[1] == 0xFF {
			order = binary.BigEndian
			b = b[2:]
		}
	}
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, order.Uint16(b[i:i+2]))
	}
	return string(utf16.Decode(u))
}

// ---- APIC (attached picture) --------------------------------------------

func parseAPIC(id string, frame []byte) (string, []byte) {
	if len(frame) < 4 {
		return "", nil
	}
	enc := frame[0]
	b := frame[1:]

	if id == "PIC" { // ID3v2.2 uses a 3 byte image format instead of a MIME type
		if len(b) < 4 {
			return "", nil
		}
		format := strings.ToLower(strings.TrimSpace(string(b[:3])))
		b = b[3:]
		b = skipEncodedString(enc, b)
		mime := "image/" + format
		if format == "jpg" || format == "jpeg" {
			mime = "image/jpeg"
		}
		return mime, b
	}

	idx := bytes.IndexByte(b, 0)
	if idx < 0 {
		return "", nil
	}
	mime := string(b[:idx])
	b = b[idx+1:]
	if len(b) < 1 {
		return mime, nil
	}
	b = b[1:] // picture type
	b = skipEncodedString(enc, b)
	return mime, b
}

// skipEncodedString removes an encoded, null terminated string and returns the rest.
func skipEncodedString(enc byte, b []byte) []byte {
	if enc == 1 || enc == 2 { // UTF-16: two-byte terminator
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return b[i+2:]
			}
		}
		return nil
	}
	idx := bytes.IndexByte(b, 0)
	if idx < 0 {
		return nil
	}
	return b[idx+1:]
}

// ---- USLT (unsynchronised lyrics) ---------------------------------------

func parseUSLT(frame []byte) string {
	if len(frame) < 4 {
		return ""
	}
	enc := frame[0]
	b := frame[1:]
	if len(b) < 3 {
		return ""
	}
	b = b[3:] // language
	b = skipEncodedString(enc, b)
	if b == nil {
		return ""
	}
	return strings.TrimRight(decodeWithEnc(enc, b), "\x00")
}

// splitEncoded splits an encoded, null terminated string (a TXXX description)
// from the value that follows it.
func splitEncoded(enc byte, b []byte) (string, string) {
	if enc == 1 || enc == 2 {
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return decodeWithEnc(enc, b[:i]), decodeWithEnc(enc, b[i+2:])
			}
		}
		return decodeWithEnc(enc, b), ""
	}
	idx := bytes.IndexByte(b, 0)
	if idx < 0 {
		return decodeWithEnc(enc, b), ""
	}
	return decodeWithEnc(enc, b[:idx]), decodeWithEnc(enc, b[idx+1:])
}

func parseTXXX(frame []byte) (string, string) {
	if len(frame) < 2 {
		return "", ""
	}
	return splitEncoded(frame[0], frame[1:])
}

func deunsync(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		out = append(out, b[i])
		if b[i] == 0xFF && i+1 < len(b) && b[i+1] == 0x00 {
			i++
		}
	}
	return out
}

// ---- duration ------------------------------------------------------------

var (
	mpeg1Bitrates = []int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}
	mpeg2Bitrates = []int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}
	mpeg1Rates    = []int{44100, 48000, 32000}
	mpeg2Rates    = []int{22050, 24000, 16000}
	mpeg25Rates   = []int{11025, 12000, 8000}
)

// frameInfo parses a Layer III frame header and returns its length in bytes,
// the number of samples it carries and the stream sample rate.
func frameInfo(h uint32) (frameLen, samples, sampleRate int, ok bool) {
	if h&0xFFE00000 != 0xFFE00000 {
		return 0, 0, 0, false
	}
	verBits := (h >> 19) & 0x3
	layerBits := (h >> 17) & 0x3
	bitrateIdx := (h >> 12) & 0xF
	rateIdx := (h >> 10) & 0x3
	padding := int((h >> 9) & 0x1)

	if layerBits != 0x1 { // only Layer III
		return 0, 0, 0, false
	}
	if bitrateIdx == 0 || bitrateIdx == 15 || rateIdx == 3 {
		return 0, 0, 0, false
	}

	var bitrates, rates []int
	version := 1
	switch verBits {
	case 0:
		version, bitrates, rates = 25, mpeg2Bitrates, mpeg25Rates
	case 2:
		version, bitrates, rates = 2, mpeg2Bitrates, mpeg2Rates
	case 3:
		version, bitrates, rates = 1, mpeg1Bitrates, mpeg1Rates
	default:
		return 0, 0, 0, false
	}

	bitrate := bitrates[bitrateIdx] * 1000
	sampleRate = rates[rateIdx]
	coef := 144
	if version != 1 {
		coef = 72
		samples = 576
	} else {
		samples = 1152
	}
	frameLen = coef*bitrate/sampleRate + padding
	if frameLen <= 4 {
		return 0, 0, 0, false
	}
	return frameLen, samples, sampleRate, true
}

// estimateDuration walks the MPEG frames and sums their sample counts. This
// works for both CBR and VBR streams.
func estimateDuration(audio []byte) float64 {
	off := 0
	for off+4 <= len(audio) {
		if audio[off] == 0xFF && audio[off+1]&0xE0 == 0xE0 {
			break
		}
		off++
	}

	totalSamples := 0
	sampleRate := 0
	for off+4 <= len(audio) {
		h := binary.BigEndian.Uint32(audio[off : off+4])
		fl, sp, sr, ok := frameInfo(h)
		if !ok {
			off++
			continue
		}
		if sampleRate == 0 {
			sampleRate = sr
		}
		totalSamples += sp
		off += fl
	}
	if sampleRate == 0 {
		return 0
	}
	return float64(totalSamples) / float64(sampleRate)
}
