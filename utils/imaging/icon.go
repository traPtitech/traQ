package imaging

import (
	"image"

	"github.com/fivenp/go-identicon"
)

const iconSize = 256

var iconSettings = qidenticon.DefaultSettings()

// GenerateIcon アイコン画像を生成します
func GenerateIcon(salt string) image.Image {
	return qidenticon.Render(qidenticon.Code(salt), iconSize, iconSettings)
}
