package model

import (
	"time"

	"gorm.io/gorm"
)

type CalendarNote struct {
	gorm.Model
	Content string    `json:"content"`
	Date    time.Time `json:"date"`
	Status  uint      `json:"status"`
}

func (cn *CalendarNote) TableName() string {
	return "calendar_notes"
}

func (cn *CalendarNote) GetContent() string {
	return cn.Content
}

func (cn *CalendarNote) GetDate() string {
	return cn.Date.String()
}

func (cn *CalendarNote) GetCreatedAt() string {
	return cn.CreatedAt.String()
}
