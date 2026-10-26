package store

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
)

const (
	maximumInputBytes     = 25 * 1024 * 1024
	maximumPixelDimension = 12_000
	maximumPixelCount     = 40_000_000
	storedPixelDimension  = 4_096
	thumbnailPixel        = 88
)

var errImageTooLarge = errors.New("image exceeds storage limits")

func prepareImage(data []byte) (pngData []byte, width, height int, err error) {
	if len(data) == 0 {
		return nil, 0, 0, errors.New("empty image")
	}
	if len(data) > maximumInputBytes {
		return nil, 0, 0, errImageTooLarge
	}

	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := source.Bounds()
	width, height = bounds.Dx(), bounds.Dy()
	if width < 1 || height < 1 ||
		width > maximumPixelDimension || height > maximumPixelDimension ||
		width*height > maximumPixelCount {
		return nil, 0, 0, errImageTooLarge
	}

	scale := 1.0
	if max(width, height) > storedPixelDimension {
		scale = float64(storedPixelDimension) / float64(max(width, height))
	}
	scaledWidth := max(int(float64(width)*scale), 1)
	scaledHeight := max(int(float64(height)*scale), 1)

	scaled := image.NewNRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), source, bounds, draw.Over, nil)

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, scaled); err != nil {
		return nil, 0, 0, err
	}
	if buffer.Len() > maximumInputBytes {
		return nil, 0, 0, errImageTooLarge
	}
	return buffer.Bytes(), scaledWidth, scaledHeight, nil
}

func thumbnail(data []byte) (image.Image, error) {
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image bounds")
	}
	scale := float64(thumbnailPixel) / float64(max(width, height))
	if scale > 1 {
		scale = 1
	}
	targetWidth := max(int(float64(width)*scale), 1)
	targetHeight := max(int(float64(height)*scale), 1)
	target := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, bounds, draw.Over, nil)
	return target, nil
}

func pngDimensions(data []byte) (int, int, bool) {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, false
	}
	return config.Width, config.Height, true
}

func safeImageName(name string) bool {
	return name != "" &&
		filepath.Base(name) == name &&
		!strings.ContainsAny(name, `/\`) &&
		strings.HasSuffix(strings.ToLower(name), ".png")
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".img-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
