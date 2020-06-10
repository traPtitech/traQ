package payload

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// TagAdded TAG_ADDEDイベントペイロード
type TagAdded struct {
	Base
	TagID uuid.UUID `json:"tagId"`
	Tag   string    `json:"tag"`
}

func MakeTagAdded(tag *model.Tag) *TagAdded {
	return &TagAdded{
		Base:  MakeBase(),
		TagID: tag.ID,
		Tag:   tag.Name,
	}
}
