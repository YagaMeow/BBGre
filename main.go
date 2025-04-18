package main

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
	"bbgre/service"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

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
		fmt.Println("[Gorm] Failed to migrate user database", err)
		return
	}
	err = global.DB.AutoMigrate(&model.Article{})
	if err != nil {
		fmt.Println("[Gorm] Failed to migrate article database", err)
		return
	}
	fmt.Println("[Gorm] Database migrated successfully")
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	public := r.Group("/api")
	{
		public.GET("/articles", service.GetArticles)
		public.GET("/articles/:id", service.GetArticle)
	}

	auth := r.Group("/api")
	auth.Use(middleware.JWTAuthMiddleware())
	{
		auth.POST("/articles", service.CreateArticle)
		auth.PUT("/articles/:id", service.UpdateArticle)
		auth.DELETE("/articles/:id", service.DeleteArticle)
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Server is running")
	})

	r.POST("/login", service.HandleLogin)

	r.Run(":8889")
}
