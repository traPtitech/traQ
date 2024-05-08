package imaging

import (
	"image"

	"github.com/motoki317/go-identicon"
)

const iconSize = 256

var iconSettings = identicon.DefaultSettings()

// GenerateIcon アイコン画像を生成します
func GenerateIcon(salt string) (image.Image, error) {
	return identicon.Render(identicon.Code(salt), iconSize, iconSettings)
}
