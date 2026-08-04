package service

import (
	"bbgre/middleware"
	"bbgre/utils"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

//	@Summary	登录
//	@Produce	json
//	@Param		request	body		LoginRequest	true	"用户名"
//	@Success	200		{object}	string			"成功"
//	@Failure	400		{object}	string			"请求错误"
//	@Failure	500		{object}	string			"内部错误"
//	@Router		/login [post]
func HandleLogin(c *gin.Context) {

	var loginData = LoginRequest{}
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
