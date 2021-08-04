package v1

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

type fileResponse struct {
	FileID      uuid.UUID `json:"fileId"`
	Name        string    `json:"name"`
	Mime        string    `json:"mime"`
	Size        int64     `json:"size"`
	MD5         string    `json:"md5"`
	HasThumb    bool      `json:"hasThumb"`
	ThumbWidth  int       `json:"thumbWidth,omitempty"`
	ThumbHeight int       `json:"thumbHeight,omitempty"`
	Datetime    time.Time `json:"datetime"`
}

func formatFile(f model.File) *fileResponse {
	hasThumb, t := f.GetThumbnail(model.ThumbnailTypeImage)
	return &fileResponse{
		FileID:      f.GetID(),
		Name:        f.GetFileName(),
		Mime:        f.GetMIMEType(),
		Size:        f.GetFileSize(),
		MD5:         f.GetMD5Hash(),
		HasThumb:    hasThumb,
		ThumbWidth:  t.Width,
		ThumbHeight: t.Height,
		Datetime:    f.GetCreatedAt(),
	}
}
