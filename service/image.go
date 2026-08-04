package service

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type ImageSize struct {
	Name     string
	MaxWidth int
	Quality  int16
}

var ImageSizes = []ImageSize{
	{"small", 400, 80},
	{"medium", 800, 85},
	{"large", 1200, 90},
	{"xlarge", 1920, 92},
}

type ImageService struct {
	UploadPath string
}

func NewImageService(uploadPath string) *ImageService {
	return &ImageService{UploadPath: uploadPath}
}

//TODO: multi-src image
