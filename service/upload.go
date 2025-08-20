package service

import (
	"bbgre/middleware"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println(err.Error())
		middleware.Error(c, 500, "Upload Failed", err.Error())
		return
	}

	fileExt := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)
	dst := filepath.Join("./uploads", newFileName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		middleware.Error(c, 500, "Save File Failed", err.Error())
		return
	}
	fileUrl := fmt.Sprintf("/uploads/%s", newFileName)

	middleware.Success(c, gin.H{
		"message": "done",
		"url":     fileUrl,
	})
}
