package utils

import (
	"github.com/jakobvarmose/go-qidenticon"
	"image"
)

const iconSize = 256

var iconSettings = qidenticon.DefaultSettings()

func GenerateIcon(salt string) image.Image {
	return qidenticon.Render(qidenticon.Code(salt), iconSize, iconSettings)
}
