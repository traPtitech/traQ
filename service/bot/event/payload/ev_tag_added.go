package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// TagAdded TAG_ADDEDイベントペイロード
type TagAdded struct {
	Base
	TagID uuid.UUID `json:"tagId"`
	Tag   string    `json:"tag"`
}

func MakeTagAdded(et time.Time, tag *model.Tag) *TagAdded {
	return &TagAdded{
		Base:  MakeBase(et),
		TagID: tag.ID,
		Tag:   tag.Name,
	}
}
