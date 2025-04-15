package service

import (
	"bbgre/global"
	"bbgre/model"
	"errors"

	"gorm.io/gorm"
)

func Login(username, password string) (*model.User, error) {
	var user model.User
	err := global.DB.Where("username = ? AND password = ?", username, password).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("invalid username or password")
	}
	return &user, err
}
