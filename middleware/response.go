package middleware

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

func SuccessMessageOnly(c *gin.Context, message string) {
	c.JSON(200, Response{
		Code:    200,
		Message: message,
		Data:    nil,
	})
}

func Error(c *gin.Context, code int, message string, errors interface{}) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
