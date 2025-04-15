package main

import (
	"bbgre/global"
	"bbgre/model"
	"bbgre/service"
	"bbgre/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("[Blog] Welcome")
	dsn := "root:root@tcp(127.0.0.1:3306)/blog?charset=utf8&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("[Gorm]", err)
		return
	} else {
		fmt.Println("[Gorm] Connected to the database successfully")
	}
	global.DB = db
	err = global.DB.AutoMigrate(&model.User{})
	if err != nil {
		fmt.Println("[Gorm] Failed to migrate the database", err)
		return
	}
	fmt.Println("[Gorm] Database migrated successfully")
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Server is running")
	})

	r.POST("/login", func(c *gin.Context) {
		var loginData struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&loginData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := service.Login(loginData.Username, loginData.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		token, err := utils.GenerateToken(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})
	r.Run()
}
