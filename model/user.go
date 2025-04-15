package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"unique;not null"`
	Password string `json:"password" gorm:"not null"`
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) GetUserId() uint {
	return u.ID
}
