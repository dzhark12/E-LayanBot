package validator

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

func ImageType(data []byte, maxBytes int64) (mime, extension string, err error) {
	if int64(len(data)) == 0 || int64(len(data)) > maxBytes {
		return "", "", fmt.Errorf("image size is invalid")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 10000 || config.Height > 10000 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return "", "", fmt.Errorf("invalid image")
	}
	switch format {
	case "jpeg":
		return "image/jpeg", ".jpg", nil
	case "png":
		return "image/png", ".png", nil
	default:
		return "", "", fmt.Errorf("unsupported image format")
	}
}
