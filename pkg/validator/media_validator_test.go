package validator

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodedImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.White)
	var data bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&data, img)
	} else {
		err = jpeg.Encode(&data, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestImageTypeRecognizesJPEGAndPNG(t *testing.T) {
	for _, tc := range []struct{ format, mime, extension string }{
		{"jpeg", "image/jpeg", ".jpg"},
		{"png", "image/png", ".png"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			data := encodedImage(t, tc.format, 2, 2)
			mime, extension, err := ImageType(data, int64(len(data)))
			if err != nil || mime != tc.mime || extension != tc.extension {
				t.Fatalf("ImageType() = %q, %q, %v", mime, extension, err)
			}
		})
	}
}

func TestImageTypeRejectsInvalidAndOversizedImages(t *testing.T) {
	valid := encodedImage(t, "png", 2, 2)
	for name, data := range map[string][]byte{
		"empty": nil, "invalid": []byte("not an image"), "too large": valid,
		"dimensions": encodedImage(t, "png", 10001, 1),
	} {
		t.Run(name, func(t *testing.T) {
			limit := int64(1 << 20)
			if name == "too large" {
				limit = int64(len(data) - 1)
			}
			if _, _, err := ImageType(data, limit); err == nil {
				t.Fatal("ImageType() accepted invalid image")
			}
		})
	}
}
