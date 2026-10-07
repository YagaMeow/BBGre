package model

import "gorm.io/gorm"

// Music stores an uploaded song and the metadata parsed from its MP3 file.
type Music struct {
	gorm.Model
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Album     string  `json:"album"`
	Year      string  `json:"year"`
	Track     string  `json:"track"`
	Genre     string  `json:"genre"`
	Composer  string  `json:"composer"`
	BPM       string  `json:"bpm"`
	Mood      string  `json:"mood"`
	Featuring string  `json:"featuring"`
	Copyright string  `json:"copyright"`
	Comment   string  `json:"comment" gorm:"type:text"`
	AudioUrl  string  `json:"audio_url"`
	CoverUrl  string  `json:"cover_url"`
	LyricsUrl string  `json:"lyrics_url"`
	Lyrics    string  `json:"-" gorm:"type:longtext"`
	Duration  float64 `json:"duration"`
	FileSize  int64   `json:"file_size"`
}

func (m *Music) TableName() string {
	return "musics"
}
