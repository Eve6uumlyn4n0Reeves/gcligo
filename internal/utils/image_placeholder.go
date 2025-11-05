package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
)

// CreateWhiteImageBase64 creates a small white PNG with the given aspect ratio (e.g., "16:9", "1:1").
// Returns base64-encoded PNG data without data: prefix.
func CreateWhiteImageBase64(aspect string) (string, error) {
	// Default 1:1
	w, h := 16, 16
	a := strings.TrimSpace(aspect)
	if a != "" && strings.Contains(a, ":") {
		parts := strings.SplitN(a, ":", 2)
		if len(parts) == 2 {
			if aw, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && aw > 0 {
				if ah, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && ah > 0 {
					// normalize to small size ~ 160px on the long edge
					long := 160
					if aw >= ah {
						w = long
						h = int(float64(long) * float64(ah) / float64(aw))
						if h < 1 {
							h = 1
						}
					} else {
						h = long
						w = int(float64(long) * float64(aw) / float64(ah))
						if w < 1 {
							w = 1
						}
					}
				}
			}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, white)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ApplyFlashImagePreviewPlaceholder mutates a Gemini request to inject a white placeholder
// image and a guidance text when targeting flash-image-preview with only aspectRatio and
// without any inlineData image. Returns true if a placeholder was injected.
func ApplyFlashImagePreviewPlaceholder(request map[string]any, model string, auto bool) bool {
	if !auto {
		return false
	}
	if strings.ToLower(model) != "gemini-2.5-flash-image-preview" {
		return false
	}
	if request == nil {
		return false
	}
	gc, _ := request["generationConfig"].(map[string]any)
	if gc == nil {
		return false
	}
	imgCfg, ok := gc["imageConfig"].(map[string]any)
	if !ok {
		return false
	}
	ar, ok := imgCfg["aspectRatio"].(string)
	if !ok || strings.TrimSpace(ar) == "" {
		return false
	}
	// detect inlineData presence
	hasInline := false
	if contents, okc := request["contents"].([]any); okc {
		for _, c0 := range contents {
			if cm, okm := c0.(map[string]any); okm {
				if parts, okp := cm["parts"].([]any); okp {
					for _, p := range parts {
						if pm, okpm := p.(map[string]any); okpm {
							if _, okID := pm["inlineData"].(map[string]any); okID {
								hasInline = true
								break
							}
						}
					}
				}
			}
			if hasInline {
				break
			}
		}
	}
	if hasInline {
		return false
	}
	b64, err := CreateWhiteImageBase64(ar)
	if err != nil || b64 == "" {
		return false
	}
	guide := map[string]any{"text": "Based on the following requirements, create an image within the uploaded picture. The new content MUST completely cover the entire area of the original picture, maintaining its exact proportions, and NO blank areas should appear."}
	placeholder := map[string]any{"inlineData": map[string]any{"mimeType": "image/png", "data": b64}}
	var contents []any
	if arr, okc := request["contents"].([]any); okc && len(arr) > 0 {
		contents = arr
	} else {
		contents = []any{map[string]any{"role": "user", "parts": []any{}}}
	}
	if first, okf := contents[0].(map[string]any); okf {
		parts, _ := first["parts"].([]any)
		newParts := []any{guide, placeholder}
		newParts = append(newParts, parts...)
		first["parts"] = newParts
		contents[0] = first
		request["contents"] = contents
		// ensure response modalities and drop imageConfig
		gc["responseModalities"] = []any{"Image", "Text"}
		delete(gc, "imageConfig")
		return true
	}
	return false
}
