package imaging

import (
	"bytes"
	"image/gif"
)

// GifToBytesReader GIF画像を*bytes.Readerに書き出します
func GifToBytesReader(src *gif.GIF) (*bytes.Reader, error) {
	buf := new(bytes.Buffer)
	if err := gif.EncodeAll(buf, src); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}
