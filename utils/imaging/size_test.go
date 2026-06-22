package imaging

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetchImageSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	imgData := buf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imgData)
		case "/not_found":
			w.WriteHeader(http.StatusNotFound)
		case "/invalid_image":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("not an image"))
		}
	}))
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("success", func(t *testing.T) {
		w, h, err := FetchImageSize(context.Background(), client, ts.URL+"/image.png")
		assert.NoError(t, err)
		assert.Equal(t, 100, w)
		assert.Equal(t, 50, h)
	})

	t.Run("not found", func(t *testing.T) {
		_, _, err := FetchImageSize(context.Background(), client, ts.URL+"/not_found")
		assert.Error(t, err)
	})

	t.Run("invalid image", func(t *testing.T) {
		_, _, err := FetchImageSize(context.Background(), client, ts.URL+"/invalid_image")
		assert.Error(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		_, _, err := FetchImageSize(context.Background(), client, "invalid-url")
		assert.Error(t, err)
	})
}
