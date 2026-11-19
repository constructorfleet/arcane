package workspace

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
)

const (
	volumeImagePreviewMaxBytes     = 5 * 1024 * 1024
	volumeImagePreviewMaxDimension = 4096
)

func volumeWorkspaceImagePreview(mimeType string, content []byte, size int64) string {
	if size <= 0 || size > volumeImagePreviewMaxBytes || int64(len(content)) != size {
		return ""
	}
	var config image.Config
	var err error
	switch mimeType {
	case "image/png":
		config, err = png.DecodeConfig(bytes.NewReader(content))
	case "image/jpeg":
		config, err = jpeg.DecodeConfig(bytes.NewReader(content))
	case "image/gif":
		config, err = gif.DecodeConfig(bytes.NewReader(content))
	default:
		return ""
	}
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > volumeImagePreviewMaxDimension || config.Height > volumeImagePreviewMaxDimension {
		return ""
	}
	return base64.StdEncoding.EncodeToString(content)
}
