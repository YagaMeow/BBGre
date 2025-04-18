package service

import (
	"bbgre/middleware"
	"bbgre/utils"

	"github.com/gin-gonic/gin"
)

func HandleLogin(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&loginData); err != nil {
		middleware.Error(c, 400, "Params Error", err.Error())
		return
	}

	user, err := Login(loginData.Username, loginData.Password)
	if err != nil {
		middleware.Error(c, 403, "Login failed", err.Error())
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		middleware.Error(c, 500, "Token generation failed", err.Error())
		return
	}
	middleware.Success(c, gin.H{
		"user":  user,
		"token": token,
	})
}
