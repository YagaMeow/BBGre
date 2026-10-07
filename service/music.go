package service

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// webPath converts an on-disk path such as "covers/jpg/x.jpg" into a URL that
// can be served by the /api static routes.
func webPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	return "/api/" + strings.TrimPrefix(p, "/")
}

// storedPath maps a served URL such as /api/uploads/audio/x.mp3 back to an
// on-disk path.
func storedPath(url string) string {
	rel := strings.TrimPrefix(url, "/api/")
	if rel == url || rel == "" {
		return ""
	}
	return filepath.Join(".", filepath.FromSlash(rel))
}

func readStoredFile(url string) []byte {
	p := storedPath(url)
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	return b
}

// baseFromAudioUrl returns the unique base name of a song (without extension).
func baseFromAudioUrl(url string) string {
	rel := strings.TrimPrefix(url, "/api/")
	if rel == url {
		return ""
	}
	name := filepath.Base(filepath.FromSlash(rel))
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// originalCoverPath is where the untouched cover bytes are kept so that the
// artwork can be re-embedded without recompression.
func originalCoverPath(base string) string {
	return filepath.Join("./covers/original", base)
}

func saveOriginalCover(base string, data []byte) {
	if base == "" || len(data) == 0 {
		return
	}
	if err := os.MkdirAll("./covers/original", 0755); err != nil {
		return
	}
	_ = os.WriteFile(originalCoverPath(base), data, 0644)
}

func imageExt(mime string, data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8:
		return ".jpg"
	case len(data) >= 8 && string(data[0:8]) == "\x89PNG\r\n\x1a\n":
		return ".png"
	case len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a"):
		return ".gif"
	}
	mime = strings.ToLower(mime)
	switch {
	case strings.Contains(mime, "png"):
		return ".png"
	case strings.Contains(mime, "gif"):
		return ".gif"
	default:
		return ".jpg"
	}
}

func musicResponse(m model.Music, withLyrics bool) gin.H {
	resp := gin.H{
		"id":         m.ID,
		"title":      m.Title,
		"artist":     m.Artist,
		"album":      m.Album,
		"year":       m.Year,
		"track":      m.Track,
		"genre":      m.Genre,
		"composer":   m.Composer,
		"bpm":        m.BPM,
		"mood":       m.Mood,
		"featuring":  m.Featuring,
		"copyright":  m.Copyright,
		"comment":    m.Comment,
		"cover_url":  m.CoverUrl,
		"audio_url":  m.AudioUrl,
		"lyrics_url": m.LyricsUrl,
		"duration":   m.Duration,
		"file_size":  m.FileSize,
		"created_at": m.CreatedAt.Format(time.RFC3339),
	}
	if withLyrics {
		lines := ParseLyrics(m.Lyrics)
		if lines == nil {
			lines = []LyricLine{}
		}
		resp["lyrics"] = lines
	}
	return resp
}

// GetMusicList returns every uploaded song (newest first).
func GetMusicList(c *gin.Context) {
	var songs []model.Music
	if err := global.DB.Order("created_at desc").Find(&songs).Error; err != nil {
		middleware.Error(c, 500, "Get music failed", err.Error())
		return
	}
	resp := make([]gin.H, 0, len(songs))
	for _, s := range songs {
		resp = append(resp, musicResponse(s, false))
	}
	middleware.Success(c, resp)
}

// GetMusicDetail returns a single song together with its parsed lyrics.
func GetMusicDetail(c *gin.Context) {
	var song model.Music
	if err := global.DB.First(&song, c.Param("id")).Error; err != nil {
		middleware.Error(c, 404, "Music not found", err.Error())
		return
	}
	middleware.Success(c, musicResponse(song, true))
}

// UploadMusic stores an MP3, parses its ID3 metadata and extracts the embedded
// cover art and lyrics.
func UploadMusic(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.Error(c, 400, "Please choose an audio file", err.Error())
		return
	}

	origExt := filepath.Ext(fileHeader.Filename)
	ext := strings.ToLower(origExt)
	if ext == "" {
		ext = ".mp3"
	}
	base := fmt.Sprintf("%d", time.Now().UnixNano())

	if err := os.MkdirAll("./uploads/audio", 0755); err != nil {
		middleware.Error(c, 500, "Create folder failed", err.Error())
		return
	}
	audioName := base + ext
	audioPath := filepath.Join("./uploads/audio", audioName)
	if err := c.SaveUploadedFile(fileHeader, audioPath); err != nil {
		middleware.Error(c, 500, "Save audio failed", err.Error())
		return
	}

	data, err := os.ReadFile(audioPath)
	if err != nil {
		middleware.Error(c, 500, "Read audio failed", err.Error())
		return
	}

	if ext == ".ncm" && !IsNCM(data) {
		_ = os.Remove(audioPath)
		middleware.Error(c, 400, "该文件不是有效的 ncm 文件", nil)
		return
	}

	// NetEase Cloud Music containers are decrypted on the server into a plain
	// MP3/FLAC stream; the temporary .ncm container is then discarded.
	var ncmMeta *NCMMeta
	var ncmCover []byte
	if IsNCM(data) {
		res, derr := DecodeNCM(data)
		if derr != nil {
			_ = os.Remove(audioPath)
			middleware.Error(c, 400, "该 ncm 文件无法解密", derr.Error())
			return
		}
		data = res.Audio
		m := res.Meta
		ncmMeta = &m
		ncmCover = res.Cover

		audioExt := ".mp3"
		if strings.EqualFold(res.Format, "flac") {
			audioExt = ".flac"
		}
		_ = os.Remove(audioPath)
		audioName = base + audioExt
		audioPath = filepath.Join("./uploads/audio", audioName)
		if werr := os.WriteFile(audioPath, data, 0644); werr != nil {
			middleware.Error(c, 500, "Save audio failed", werr.Error())
			return
		}
	}

	info := &mp3Info{}
	if strings.HasSuffix(strings.ToLower(audioName), ".mp3") {
		info = ParseMP3(data)
	}

	// Cover art: embedded ID3 artwork by default, the .ncm container artwork
	// wins when present.
	picture := info.Picture
	pictureMIME := info.PictureMIME
	if len(ncmCover) > 0 {
		picture = ncmCover
		pictureMIME = DetectImageMIME(ncmCover)
	}
	coverURL := ""
	if len(picture) > 0 {
		saveOriginalCover(base, picture)
		coverName := base + imageExt(pictureMIME, picture)
		if p, err := SizeHandler(bytes.NewReader(picture), coverName, "./covers"); err == nil {
			coverURL = webPath(p)
		} else {
			fmt.Println("[Music] cover parse failed:", err)
		}
	}

	// Lyrics: a separately uploaded .lrc wins over embedded lyrics.
	lyricsText := info.Lyrics
	if lrcHeader, lerr := c.FormFile("lyric"); lerr == nil && lrcHeader != nil {
		if lf, oerr := lrcHeader.Open(); oerr == nil {
			if b, rerr := io.ReadAll(lf); rerr == nil {
				lyricsText = string(b)
			}
			lf.Close()
		}
	}
	lyricsLines := ParseLyrics(lyricsText)
	normalizedLyrics := LyricsToLRC(lyricsLines)
	lyricsURL := ""
	if len(lyricsLines) > 0 {
		lrcName := base + ".lrc"
		if err := os.WriteFile(filepath.Join("./uploads/audio", lrcName), []byte(normalizedLyrics), 0644); err == nil {
			lyricsURL = "/api/uploads/audio/" + lrcName
		}
	}

	title := strings.TrimSpace(info.Title)
	artist := strings.TrimSpace(info.Artist)
	album := strings.TrimSpace(info.Album)
	duration := info.Duration
	if ncmMeta != nil {
		if title == "" {
			title = strings.TrimSpace(ncmMeta.MusicName)
		}
		if artist == "" {
			artist = strings.TrimSpace(ncmMeta.ArtistNames())
		}
		if album == "" {
			album = strings.TrimSpace(ncmMeta.Album)
		}
		if duration <= 0 && ncmMeta.Duration > 0 {
			duration = float64(ncmMeta.Duration) / 1000
		}
	}
	if title == "" {
		title = strings.TrimSuffix(fileHeader.Filename, origExt)
	}

	song := model.Music{
		Title:     title,
		Artist:    artist,
		Album:     album,
		Year:      strings.TrimSpace(info.Year),
		Track:     strings.TrimSpace(info.Track),
		Genre:     strings.TrimSpace(info.Genre),
		Composer:  strings.TrimSpace(info.Composer),
		BPM:       strings.TrimSpace(info.BPM),
		Mood:      strings.TrimSpace(info.Mood),
		Featuring: strings.TrimSpace(info.Featuring),
		Comment:   strings.TrimSpace(info.Comment),
		AudioUrl:  "/api/uploads/audio/" + audioName,
		CoverUrl:  coverURL,
		LyricsUrl: lyricsURL,
		Lyrics:    normalizedLyrics,
		Duration:  duration,
		FileSize:  int64(len(data)),
	}
	if err := global.DB.Create(&song).Error; err != nil {
		middleware.Error(c, 500, "Save music failed", err.Error())
		return
	}
	middleware.Success(c, musicResponse(song, true))
}

// DownloadMusic serves the original audio file as a browser download.
func DownloadMusic(c *gin.Context) {
	var song model.Music
	if err := global.DB.First(&song, c.Param("id")).Error; err != nil {
		middleware.Error(c, 404, "Music not found", err.Error())
		return
	}
	p := storedPath(song.AudioUrl)
	if p == "" {
		middleware.Error(c, 404, "Audio file not found", nil)
		return
	}
	if _, err := os.Stat(p); err != nil {
		middleware.Error(c, 404, "Audio file not found", err.Error())
		return
	}
	c.FileAttachment(p, downloadFileName(song))
}

func downloadFileName(song model.Music) string {
	name := strings.TrimSpace(song.Title)
	if name == "" {
		name = "music"
	}
	if artist := strings.TrimSpace(song.Artist); artist != "" {
		name += " - " + artist
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = strings.TrimSpace(replacer.Replace(name))
	if name == "" {
		name = "music"
	}
	ext := filepath.Ext(song.AudioUrl)
	if ext == "" {
		ext = ".mp3"
	}
	return name + ext
}
// UpdateMusic edits a song's metadata (and optionally its cover/lyrics) and
// writes the new tags back into the MP3 file.
func UpdateMusic(c *gin.Context) {
	var song model.Music
	if err := global.DB.First(&song, c.Param("id")).Error; err != nil {
		middleware.Error(c, 404, "Music not found", err.Error())
		return
	}

	base := baseFromAudioUrl(song.AudioUrl)
	if base == "" {
		base = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	setIfPresent := func(name string, dst *string) {
		if v, ok := c.GetPostForm(name); ok {
			*dst = strings.TrimSpace(v)
		}
	}
	setIfPresent("title", &song.Title)
	setIfPresent("artist", &song.Artist)
	setIfPresent("album", &song.Album)
	setIfPresent("year", &song.Year)
	setIfPresent("track", &song.Track)
	setIfPresent("genre", &song.Genre)
	setIfPresent("composer", &song.Composer)
	setIfPresent("bpm", &song.BPM)
	setIfPresent("mood", &song.Mood)
	setIfPresent("featuring", &song.Featuring)
	setIfPresent("copyright", &song.Copyright)
	setIfPresent("comment", &song.Comment)

	// Lyrics: the textarea is authoritative, an uploaded .lrc overrides it.
	lyricsText := song.Lyrics
	if v, ok := c.GetPostForm("lyrics"); ok {
		lyricsText = v
	}
	if lrcHeader, lerr := c.FormFile("lyric"); lerr == nil && lrcHeader != nil {
		if lf, oerr := lrcHeader.Open(); oerr == nil {
			if b, rerr := io.ReadAll(lf); rerr == nil {
				lyricsText = string(b)
			}
			lf.Close()
		}
	}
	lyricsLines := ParseLyrics(lyricsText)
	song.Lyrics = LyricsToLRC(lyricsLines)
	if len(lyricsLines) > 0 {
		lrcName := base + ".lrc"
		if err := os.WriteFile(filepath.Join("./uploads/audio", lrcName), []byte(song.Lyrics), 0644); err == nil {
			song.LyricsUrl = "/api/uploads/audio/" + lrcName
		}
	} else {
		song.LyricsUrl = ""
	}

	// Cover: a new upload wins, otherwise re-embed the stored original bytes.
	var coverBytes []byte
	if coverHeader, cerr := c.FormFile("cover"); cerr == nil && coverHeader != nil {
		if cf, oerr := coverHeader.Open(); oerr == nil {
			coverBytes, _ = io.ReadAll(cf)
			cf.Close()
		}
	}
	if len(coverBytes) > 0 {
		if p, err := SizeHandler(bytes.NewReader(coverBytes), base+imageExt(DetectImageMIME(coverBytes), coverBytes), "./covers"); err == nil {
			song.CoverUrl = webPath(p)
			saveOriginalCover(base, coverBytes)
		} else {
			middleware.Error(c, 400, "Invalid cover image", err.Error())
			return
		}
	} else if b, err := os.ReadFile(originalCoverPath(base)); err == nil {
		coverBytes = b
	} else if song.CoverUrl != "" {
		coverBytes = readStoredFile(song.CoverUrl)
	}

	audioPath := storedPath(song.AudioUrl)
	if audioPath == "" {
		middleware.Error(c, 500, "Audio file path is invalid", nil)
		return
	}
	tags := ID3Tags{
		Title:     song.Title,
		Artist:    song.Artist,
		Album:     song.Album,
		Year:      song.Year,
		Track:     song.Track,
		Genre:     song.Genre,
		Composer:  song.Composer,
		BPM:       song.BPM,
		Mood:      song.Mood,
		Featuring: song.Featuring,
		Copyright: song.Copyright,
		Comment:   song.Comment,
		Lyrics:    song.Lyrics,
		Cover:     coverBytes,
		CoverMIME: DetectImageMIME(coverBytes),
	}
	// ID3 tags can only be written into MP3 streams; FLAC keeps its DB metadata.
	if strings.EqualFold(filepath.Ext(audioPath), ".mp3") {
		if err := WriteID3(audioPath, tags); err != nil {
			middleware.Error(c, 500, "Write tags failed", err.Error())
			return
		}
	}
	if fi, err := os.Stat(audioPath); err == nil {
		song.FileSize = fi.Size()
	}

	if err := global.DB.Save(&song).Error; err != nil {
		middleware.Error(c, 500, "Update music failed", err.Error())
		return
	}
	middleware.Success(c, musicResponse(song, true))
}

// DeleteMusic removes a song, its lyrics and its artwork.
func DeleteMusic(c *gin.Context) {
	var song model.Music
	if err := global.DB.First(&song, c.Param("id")).Error; err != nil {
		middleware.Error(c, 404, "Music not found", err.Error())
		return
	}
	removeStoredFile(song.AudioUrl)
	removeStoredFile(song.LyricsUrl)
	removeStoredFile(song.CoverUrl)
	if base := baseFromAudioUrl(song.AudioUrl); base != "" {
		_ = os.Remove(originalCoverPath(base))
	}
	if err := global.DB.Delete(&song).Error; err != nil {
		middleware.Error(c, 500, "Delete music failed", err.Error())
		return
	}
	middleware.SuccessMessageOnly(c, "Music deleted successfully")
}

func removeStoredFile(url string) {
	p := storedPath(url)
	if p == "" {
		return
	}
	_ = os.Remove(p)
}