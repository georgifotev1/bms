package main

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type ImageInput struct {
	URL  string
	File *multipart.FileHeader
}

func getImageInput(r *http.Request, formKey, imageUrl string) *ImageInput {
	imageInput := &ImageInput{}

	if imageUrl != "" {
		imageInput.URL = imageUrl
	} else {
		if _, fileHeader, err := r.FormFile(formKey); err == nil {
			imageInput.File = fileHeader
		}
	}
	return imageInput
}

func (app *application) saveImageToCloudinary(file multipart.File) (string, error) {
	ctx := context.Background()
	publicID := fmt.Sprintf("bms/%d_%s", time.Now().UnixNano(), generateSubstring(8))

	uploadResult, err := app.imageService.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "bms",
		ResourceType:   "image",
		Transformation: "q_auto,f_auto",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to Cloudinary: %w", err)
	}

	return uploadResult.SecureURL, nil
}

func (app *application) ProcessImage(img *ImageInput) (string, error) {
	if img.URL != "" {
		if _, err := url.Parse(img.URL); err != nil {
			return "", fmt.Errorf("invalid image URL: %v", err)
		}
		return img.URL, nil
	}

	if img.File != nil {
		file, err := img.File.Open()
		if err != nil {
			return "", err
		}
		defer file.Close()

		return app.saveImageToCloudinary(file)
	}

	return "", nil
}
