package service

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
	"bbgre/utils"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateArticle(c *gin.Context) {
	userID, _ := c.Get("userID")

	var input struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Uri     string `json:"uri" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Paramas Error", err.Error())
		return
	}

	article := model.Article{
		Title:    input.Title,
		Content:  input.Content,
		Uri:      input.Uri,
		AuthorId: userID.(uint),
	}

	if hasCreated := global.DB.Where("uri = ?", article.Uri).First(&model.Article{}).RowsAffected; hasCreated > 0 {
		middleware.Error(c, 400, "Article with this URI already exists", nil)
		return
	}

	if err := global.DB.Create(&article).Error; err != nil {
		middleware.Error(c, 500, "Create article failed", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"created_at": article.CreatedAt.Format(time.RFC3339),
	})

}

func UpdateArticle(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found", err.Error())
		return
	}
	if article.GetAuthorId() != userID.(uint) {
		middleware.Error(c, 403, "You are not the author of this article", nil)
		return
	}

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Paramas Error", err.Error())
		return
	}

	updates := make(map[string]interface{})
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.Content != "" {
		updates["content"] = input.Content
	}

	if err := global.DB.Model(&article).Updates(updates).Error; err != nil {
		middleware.Error(c, 500, "Update article failed", err.Error())
		return
	}

	middleware.SuccessMessageOnly(c, "Article updated successfully")

}

func UpdateArticleByUri(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleUri := c.Param("uri")

	var article model.Article
	if err := global.DB.Where("uri = ?", articleUri).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found", err.Error())
		return
	}
	if article.GetAuthorId() != userID.(uint) {
		middleware.Error(c, 403, "You are not the author of this article", nil)
		return
	}

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Paramas Error", err.Error())
		return
	}

	updates := make(map[string]interface{})
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.Content != "" {
		updates["content"] = input.Content
	}

	if err := global.DB.Model(&article).Updates(updates).Error; err != nil {
		middleware.Error(c, 500, "Update article failed", err.Error())
		return
	}

	middleware.SuccessMessageOnly(c, "Article updated successfully")
}

func DeleteArticle(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found", err.Error())
		return
	}

	if article.GetAuthorId() != userID.(uint) {
		middleware.Error(c, 403, "You are not the author of this article", nil)
		return
	}

	if err := global.DB.Delete(&article).Error; err != nil {
		middleware.Error(c, 500, "Delete article failed", err.Error())
		return
	}

	middleware.SuccessMessageOnly(c, "Article deleted successfully")

}

func GetArticles(c *gin.Context) {
	var articles []model.Article
	if err := global.DB.Order("created_at desc").Find(&articles).Error; err != nil {
		middleware.Error(c, 500, "Get articles failed", err.Error())
		return
	}

	var response []gin.H
	for _, article := range articles {
		response = append(response, gin.H{
			"id":         article.ID,
			"title":      article.Title,
			"uri":        article.Uri,
			"created_at": article.CreatedAt.Format(time.RFC3339),
			"tag":        article.Tags,
			"cover": gin.H{
				"cover_url": article.CoverUrl,
				"height":    article.CoverH,
				"width":     article.CoverW,
			},
		})
	}

	if response == nil {
		response = []gin.H{}
	}

	middleware.Success(c, response)

}

func GetArticle(c *gin.Context) {
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.First(&article, articleID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
		"tags":       article.Tags,
	})

}

func GetArticleByUri(c *gin.Context) {
	uri := c.Param("uri")

	var article model.Article
	if err := global.DB.Where("uri = ?", uri).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
		"tags":       article.Tags,
	})
}

func DeleteArticleByUri(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleUri := c.Param("uri")

	var article model.Article
	if err := global.DB.Where("uri = ?", articleUri).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found", err.Error())
		return
	}

	if article.GetAuthorId() != userID.(uint) {
		middleware.Error(c, 403, "You are not the author of this article", nil)
		return
	}

	if err := global.DB.Delete(&article).Error; err != nil {
		middleware.Error(c, 500, "Delete article failed", err.Error())
		return
	}

	middleware.SuccessMessageOnly(c, "Article deleted successfully")
}

func AddTagToArticle(c *gin.Context) {
	// userID, _ := c.Get("userID")
	var input struct {
		ID  uint   `json:"id"`
		Tag string `json:"tag"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Paramas Error", err.Error())
		return
	}

	var article model.Article
	var Tag model.Tag
	if err := global.DB.Where("name = ?", input.Tag).First(&Tag).Error; err != nil {
		Tag = model.Tag{
			Name: input.Tag,
		}
		if err := global.DB.Create(&Tag).Error; err != nil {
			middleware.Error(c, 500, "Create tag failed.", err.Error())
			return
		}
	}
	if err := global.DB.Where("id = ?", input.ID).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found.", err.Error())
		return
	}

	if err := global.DB.Model(&article).Association("Tags").Append(&Tag); err != nil {
		middleware.Error(c, 500, "Add tag failed.", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
		"tags":       article.Tags,
	})

}

func RemoveTagFromArticle(c *gin.Context) {
	var input struct {
		ID  uint   `json:"id"`
		Tag string `json:"tag"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Params error.", err.Error())
		return
	}

	var tag model.Tag
	if err := global.DB.Where("name = ?", input.Tag).First(&tag).Error; err != nil {
		middleware.Error(c, 404, "Tag not found.", err.Error())
		return
	}
	var article model.Article
	if err := global.DB.Where("id = ?", input.ID).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found.", err.Error())
		return
	}

	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 删除文章与标签的关联
	if err := tx.Model(&article).Association("Tags").Delete(&tag); err != nil {
		tx.Rollback()
		fmt.Println(err.Error())
		middleware.Error(c, 500, "Delete tag failed.", err.Error())
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// 检查该标签是否还被其他文章使用
	count := tx.Model(&tag).Association("Articles").Count()

	// 如果没有其他文章使用该标签，则删除标签
	if count == 0 {
		if err := tx.Delete(&tag).Error; err != nil {
			tx.Rollback()
			middleware.Error(c, 500, "Failed to delete unused tag", err.Error())
			return
		}
	}

	// 重新加载文章及其标签
	if err := tx.Preload("Tags").First(&article, article.ID).Error; err != nil {
		tx.Rollback()
		middleware.Error(c, 500, "Failed to load article with tags", err.Error())
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		middleware.Error(c, 500, "Transaction commit failed", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
		"tags":       article.Tags,
	})
}

func UploadCover(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	uri := c.PostForm("uri")

	var article model.Article
	if err := global.DB.Model(&model.Article{}).Where("uri = ?", uri).First(&article).Error; err != nil {
		middleware.Error(c, 404, "Article not found.", err)
		return
	}

	if err != nil {
		middleware.Error(c, 500, "Upload Failed", err.Error())
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		middleware.Error(c, 500, "Server Error", err.Error())
		return
	}
	defer file.Close()
	fileExt := filepath.Ext(fileHeader.Filename)
	newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)
	dst := filepath.Join("./covers", newFileName)

	if err := SizeHandler(file, newFileName, "./covers"); err != nil {
		middleware.Error(c, 500, "Save File Failed", err.Error())
		return
	}

	width, height, err := utils.GetImageDimensions(dst)
	if err != nil {
		fmt.Println("Failed to parse image", err)
		width = 0
		height = 0
	}

	fileUrl := fmt.Sprintf("/covers/%s", newFileName)
	article.CoverUrl = fileUrl

	updates := make(map[string]interface{})
	updates["cover_url"] = fileUrl
	updates["cover_h"] = height
	updates["cover_w"] = width
	if err := global.DB.Model(&article).Updates(updates).Error; err != nil {
		middleware.Error(c, 500, "Update cover failed", err.Error())
		return
	}
	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
		"tags":       article.Tags,
		"cover": gin.H{
			"cover_url": article.CoverUrl,
			"height":    article.CoverH,
			"width":     article.CoverW,
		},
	})
}
