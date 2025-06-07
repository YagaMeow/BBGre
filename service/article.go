package service

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
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
	if err := global.DB.Find(&articles).Error; err != nil {
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
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}

	middleware.Success(c, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
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
