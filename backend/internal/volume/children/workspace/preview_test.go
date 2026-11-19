package workspace

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"getarcane.dev/types/v3"
)

func TestVolumeWorkspaceImagePreview(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for _, format := range []string{"png", "jpeg", "gif"} {
		t.Run(format, func(t *testing.T) {
			var encoded bytes.Buffer
			switch format {
			case "png":
				require.NoError(t, png.Encode(&encoded, img))
			case "jpeg":
				require.NoError(t, jpeg.Encode(&encoded, img, nil))
			case "gif":
				require.NoError(t, gif.Encode(&encoded, img, nil))
			}
			content := encoded.Bytes()
			response, err := volumeWorkspaceFileContentResponse("preview."+format, "regular", int64(len(content)), content, volumeImagePreviewMaxBytes)
			require.NoError(t, err)
			require.False(t, response.Editable)
			require.Equal(t, types.FileReadOnlyBinary, response.ReadOnlyReason)
			require.Empty(t, response.Content)
			decoded, err := base64.StdEncoding.DecodeString(response.PreviewData)
			require.NoError(t, err)
			require.Equal(t, content, decoded)
			require.Equal(t, http.DetectContentType(content), response.MimeType)

			response, err = volumeWorkspaceFileContentResponse("preview."+format, "regular", int64(len(content)), content, 1)
			require.NoError(t, err)
			require.Equal(t, types.FileReadOnlyTooLarge, response.ReadOnlyReason)
			require.Empty(t, response.PreviewData)
		})
	}
}

func TestVolumeWorkspaceImagePreviewLimits(t *testing.T) {
	var valid bytes.Buffer
	require.NoError(t, png.Encode(&valid, image.NewNRGBA(image.Rect(0, 0, 32, 16))))
	large := make([]byte, volumeImagePreviewMaxBytes+1)
	copy(large, valid.Bytes())
	require.Empty(t, volumeWorkspaceImagePreview("image/png", large, int64(len(large))))
	require.Empty(t, volumeWorkspaceImagePreview("image/png", valid.Bytes(), int64(valid.Len()+1)))
	require.Empty(t, volumeWorkspaceImagePreview("image/png", []byte("not a png"), 9))
	require.Empty(t, volumeWorkspaceImagePreview("image/svg+xml", []byte("<svg/>"), 6))
	require.Empty(t, volumeWorkspaceImagePreview("text/html", []byte("<h1>test</h1>"), 13))

	var atLimit bytes.Buffer
	require.NoError(t, png.Encode(&atLimit, image.NewNRGBA(image.Rect(0, 0, volumeImagePreviewMaxDimension, 1))))
	require.NotEmpty(t, volumeWorkspaceImagePreview("image/png", atLimit.Bytes(), int64(atLimit.Len())))
	var aboveLimit bytes.Buffer
	require.NoError(t, png.Encode(&aboveLimit, image.NewNRGBA(image.Rect(0, 0, volumeImagePreviewMaxDimension+1, 1))))
	require.Empty(t, volumeWorkspaceImagePreview("image/png", aboveLimit.Bytes(), int64(aboveLimit.Len())))
}
