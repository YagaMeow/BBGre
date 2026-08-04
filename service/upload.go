package service

import (
	"bbgre/middleware"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	webp "github.com/bep/webptemp"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
)

var sizes = []int{400, 800, 1200}

func UploadHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		fmt.Println(err.Error())
		middleware.Error(c, 500, "Upload Failed", err.Error())
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		fmt.Println(err.Error())
		middleware.Error(c, 500, "Server Error", err.Error())
		return
	}
	defer file.Close()

	fileExt := filepath.Ext(fileHeader.Filename)
	newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileExt)

	if _, err := SizeHandler(file, newFileName, "./uploads"); err != nil {
		middleware.Error(c, 500, "Save File Failed", err.Error())
		return
	}

	// if err := c.SaveUploadedFile(file, dst); err != nil {
	// 	middleware.Error(c, 500, "Save File Failed", err.Error())
	// 	return
	// }
	filePath := fmt.Sprintf("/uploads/webp/%s", strings.TrimSuffix(newFileName, fileExt))

	middleware.Success(c, gin.H{
		"message": "done",
		"url": gin.H{
			"samll":  filePath + "-400w.webp",
			"middle": filePath + "-800w.webp",
			"large":  filePath + "-1200w.webp",
		},
	})
}

func SizeHandler(file io.Reader, filename string, path string) (string, error) {
	srcImg, err := imaging.Decode(file)
	if err != nil {
		return "", err
	}
	basename := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, w := range sizes {
		dstImg := imaging.Resize(srcImg, w, 0, imaging.Lanczos)
		jpegPath := filepath.Join(path+"/jpg", fmt.Sprintf("%s-%dw.jpg", basename, w))
		jpgDir := filepath.Dir(jpegPath)
		if err := os.MkdirAll(jpgDir, 0755); err != nil {
			return "", fmt.Errorf("Create folder failed: %w", err)
		}
		err := imaging.Save(dstImg, jpegPath, imaging.JPEGQuality(85))
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		webpPath := filepath.Join(path+"/webp", fmt.Sprintf("%s-%dw.webp", basename, w))
		webpDir := filepath.Dir(webpPath)
		if err := os.MkdirAll(webpDir, 0755); err != nil {
			return "", fmt.Errorf("Create folder failed: %w", err)
		}
		if err := saveAsWebp(dstImg, webpPath, 80); err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	return filepath.Join(path+"/jpg", fmt.Sprintf("%s-%dw.jpg", basename, 1200)), nil
}

func saveAsWebp(img image.Image, path string, quality float32) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return webp.Encode(out, img, webp.Options{Quality: int(quality)})
}
