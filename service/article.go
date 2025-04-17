package service

import (
	"bbgre/global"
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
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	article := model.Article{
		Title:    input.Title,
		Content:  input.Content,
		Uri:      input.Uri,
		AuthorId: userID.(uint),
	}

	if err := global.DB.Create(&article).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"created_at": article.CreatedAt.Format(time.RFC3339)})

}

func UpdateArticle(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}
	if article.GetAuthorId() != userID.(uint) {
		c.JSON(403, gin.H{"error": "You are not the author of this article"})
		return
	}

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
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
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Article updated successfully"})

}

func DeleteArticle(c *gin.Context) {
	userID, _ := c.Get("userID")
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}

	if article.GetAuthorId() != userID.(uint) {
		c.JSON(403, gin.H{"error": "You are not the author of this article"})
		return
	}

	if err := global.DB.Delete(&article).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Article deleted successfully"})

}

func GetArticles(c *gin.Context) {
	var articles []model.Article
	if err := global.DB.Find(&articles).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
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

	c.JSON(200, response)

}

func GetArticle(c *gin.Context) {
	articleID := c.Param("id")

	var article model.Article
	if err := global.DB.Where("id = ?", articleID).First(&article).Error; err != nil {
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}

	c.JSON(200, gin.H{
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
		c.JSON(404, gin.H{"error": "Article not found"})
		return
	}

	c.JSON(200, gin.H{
		"id":         article.ID,
		"title":      article.Title,
		"uri":        article.Uri,
		"content":    article.Content,
		"created_at": article.CreatedAt.Format(time.RFC3339),
	})
}
