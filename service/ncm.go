package service

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// NetEase Cloud Music (.ncm) container keys.
var (
	ncmCoreKey = []byte{0x68, 0x7A, 0x48, 0x52, 0x41, 0x6D, 0x73, 0x6F, 0x35, 0x6B, 0x49, 0x6E, 0x62, 0x61, 0x78, 0x57}
	ncmMetaKey = []byte{0x23, 0x31, 0x34, 0x6C, 0x6A, 0x6B, 0x5F, 0x21, 0x5C, 0x5D, 0x26, 0x30, 0x55, 0x3C, 0x27, 0x28}
)

const ncmKeyPrefix = "neteasecloudmusic"

// NCMMeta is the JSON metadata embedded in an .ncm container.
type NCMMeta struct {
	MusicName string          `json:"musicName"`
	Artist    [][]interface{} `json:"artist"`
	Album     string          `json:"album"`
	Format    string          `json:"format"`
	Bitrate   int             `json:"bitrate"`
	Duration  int             `json:"duration"`
}

// ArtistNames flattens the [[name, id], ...] artist list into a single string.
func (m NCMMeta) ArtistNames() string {
	names := make([]string, 0, len(m.Artist))
	for _, entry := range m.Artist {
		if len(entry) == 0 {
			continue
		}
		if name, ok := entry[0].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "/")
}

// NCMResult holds the decrypted contents of an .ncm file.
type NCMResult struct {
	Audio     []byte
	Format    string
	Cover     []byte
	CoverMIME string
	Meta      NCMMeta
}

// IsNCM reports whether the given bytes look like an .ncm container.
func IsNCM(data []byte) bool {
	return len(data) >= 8 && string(data[0:8]) == "CTENFDAM"
}

// DecodeNCM decrypts an .ncm container into the raw audio stream (mp3/flac),
// the embedded cover image and the JSON metadata.
func DecodeNCM(data []byte) (*NCMResult, error) {
	r := bytes.NewReader(data)
	magic := make([]byte, 8)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	if string(magic) != "CTENFDAM" {
		return nil, errors.New("不是有效的 ncm 文件")
	}
	if _, err := r.Seek(2, io.SeekCurrent); err != nil {
		return nil, err
	}

	// --- decrypt the RC4 key box ---
	keyLen, err := ncmReadUint32(r)
	if err != nil {
		return nil, err
	}
	keyData := make([]byte, keyLen)
	if _, err := io.ReadFull(r, keyData); err != nil {
		return nil, err
	}
	for i := range keyData {
		keyData[i] ^= 0x64
	}
	keyData, err = ncmAESDecrypt(ncmCoreKey, keyData)
	if err != nil {
		return nil, err
	}
	keyData = ncmUnpad(keyData)
	if !bytes.HasPrefix(keyData, []byte(ncmKeyPrefix)) {
		return nil, errors.New("ncm 密钥解析失败")
	}
	keyBox := ncmBuildKeyBox(keyData[len(ncmKeyPrefix):])

	// --- decrypt the metadata ---
	metaLen, err := ncmReadUint32(r)
	if err != nil {
		return nil, err
	}
	metaData := make([]byte, metaLen)
	if _, err := io.ReadFull(r, metaData); err != nil {
		return nil, err
	}
	for i := range metaData {
		metaData[i] ^= 0x63
	}
	if len(metaData) > 22 {
		metaData = metaData[22:]
	}
	decoded, err := base64.StdEncoding.DecodeString(string(metaData))
	if err != nil {
		return nil, err
	}
	decoded, err = ncmAESDecrypt(ncmMetaKey, decoded)
	if err != nil {
		return nil, err
	}
	decoded = ncmUnpad(decoded)
	if len(decoded) > 6 {
		decoded = decoded[6:]
	}
	var meta NCMMeta
	_ = json.Unmarshal(decoded, &meta)

	// --- skip the CRC + gap, then read the cover image ---
	if _, err := r.Seek(9, io.SeekCurrent); err != nil {
		return nil, err
	}
	coverLen, err := ncmReadUint32(r)
	if err != nil {
		return nil, err
	}
	cover := make([]byte, coverLen)
	if coverLen > 0 {
		if _, err := io.ReadFull(r, cover); err != nil {
			return nil, err
		}
	}

	// --- decrypt the audio stream ---
	audio, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(audio); i++ {
		j := byte(i + 1)
		a := int(keyBox[j])
		b := int(keyBox[(a+int(j))&0xff])
		audio[i] ^= keyBox[(a+b)&0xff]
	}

	format := strings.ToLower(strings.TrimSpace(meta.Format))
	if format == "" {
		format = detectAudioFormat(audio)
	}
	return &NCMResult{
		Audio:     audio,
		Format:    format,
		Cover:     cover,
		CoverMIME: DetectImageMIME(cover),
		Meta:      meta,
	}, nil
}

func ncmReadUint32(r io.Reader) (int, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint32(b[:])), nil
}

func ncmAESDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, errors.New("ncm 加密数据长度错误")
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Decrypt(out[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
	return out, nil
}

func ncmUnpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > len(data) || pad > aes.BlockSize {
		return data
	}
	return data[:len(data)-pad]
}

func ncmBuildKeyBox(key []byte) [256]byte {
	var box [256]byte
	for i := 0; i < 256; i++ {
		box[i] = byte(i)
	}
	if len(key) == 0 {
		return box
	}
	var c, lastByte byte
	offset := 0
	for i := 0; i < 256; i++ {
		swap := box[i]
		c = swap + lastByte + key[offset]
		offset++
		if offset >= len(key) {
			offset = 0
		}
		box[i] = box[c]
		box[c] = swap
		lastByte = c
	}
	return box
}

func detectAudioFormat(data []byte) string {
	switch {
	case len(data) >= 4 && string(data[0:4]) == "fLaC":
		return "flac"
	case len(data) >= 3 && string(data[0:3]) == "ID3":
		return "mp3"
	case len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0:
		return "mp3"
	default:
		return "mp3"
	}
}