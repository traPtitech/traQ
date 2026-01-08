package imaging

import (
	"context"
	"fmt"
	"image"

	"net/http"

	// add gif, jpeg, png support
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// add webp support
	_ "golang.org/x/image/webp"
)

// FetchImageSize 指定されたURLの画像サイズを取得
func FetchImageSize(ctx context.Context, client *http.Client, imageURL string) (width, height int, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to fetch image: %s (status code: %d)", imageURL, resp.StatusCode)
	}

	// ヘッダー部分だけを解釈してサイズを得る
	cfg, _, err := image.DecodeConfig(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	return cfg.Width, cfg.Height, nil
}
