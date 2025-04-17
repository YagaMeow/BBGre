package model

import "gorm.io/gorm"

type Article struct {
	gorm.Model
	Uri      string `json:"uri" gorm:"unique;not null"`
	Title    string `json:"title" gorm:"not null"`
	Content  string `json:"content" gorm:"not null"`
	AuthorId uint   `json:"author_id"`
}

func (a *Article) TableName() string {
	return "articles"
}

func (a *Article) GetContent() string {
	return a.Content
}

func (a *Article) GetTitle() string {
	return a.Title
}

func (a *Article) GetUri() string {
	return a.Uri
}

func (a *Article) GetId() uint {
	return a.ID
}

func (a *Article) GetCreatedAt() string {
	return a.CreatedAt.String()
}

func (a *Article) GetAuthorId() uint {
	return a.AuthorId
}
