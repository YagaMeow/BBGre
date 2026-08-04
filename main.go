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

	_ "bbgre/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	fmt.Println("[Blog] Welcome")
	dsn := "root:root@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
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
	err = global.DB.AutoMigrate(&model.Article{}, &model.Tag{}, &model.CalendarNote{})
	if err != nil {
		fmt.Println("[Gorm] Failed to migrate article database", err)
		return
	}
	fmt.Println("[Gorm] Database migrated successfully")
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "x-token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	public := r.Group("/api")
	{
		public.GET("/articles", service.GetArticles)
		public.GET("/articles/:id", service.GetArticle)
		public.GET("/articles/uri/:uri", service.GetArticleByUri)
		public.Static("/uploads", "./uploads")
		public.Static("/covers", "./covers")
		public.POST("/notes", service.GetNoteList)
	}

	auth := r.Group("/api")
	auth.Use(middleware.JWTAuthMiddleware())
	{
		articles := auth.Group("/articles")
		{
			articles.POST("/", service.CreateArticle)
			articles.PUT("/:id", service.UpdateArticle)
			articles.PUT("/uri/:uri", service.UpdateArticleByUri)
			articles.DELETE("/uri/:uri", service.DeleteArticleByUri)
			articles.POST("/cover", service.UploadCover)
		}
		notes := auth.Group("/notes")
		{
			notes.POST("/", service.CreateNote)
			notes.POST("/update", service.UpdateNote)
			notes.DELETE("/update", service.DeleteNote)
		}
		auth.POST("/auth", service.AuthorizeUser)
		auth.POST("/upload", service.UploadHandler)
		auth.POST("/addtag", service.AddTagToArticle)
		auth.POST("/removetag", service.RemoveTagFromArticle)
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Server is running")
	})

	r.POST("/login", service.HandleLogin)

	r.Run(":8889")
}
