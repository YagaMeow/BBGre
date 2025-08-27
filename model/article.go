package model

import (
	"gorm.io/gorm"
)

type Article struct {
	gorm.Model
	Uri      string `json:"uri" gorm:"type:varchar(255);not null"`
	Title    string `json:"title" gorm:"not null"`
	Content  string `json:"content" gorm:"not null"`
	AuthorId uint   `json:"author_id"`
	Tags     []Tag  `json:"tags" gorm:"many2many:article_tags;"`
}

type Tag struct {
	gorm.Model
	Name     string    `json:"name" gorm:"size:255;not null"`
	Articles []Article `json:"articles" gorm:"many2many:article_tags;"`
}

func (t *Tag) TableName() string {
	return "tags"
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
