package service

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
	"errors"

	"github.com/gin-gonic/gin"
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

func AuthorizeUser(c *gin.Context) {
	middleware.SuccessMessageOnly(c, "Success")
}
