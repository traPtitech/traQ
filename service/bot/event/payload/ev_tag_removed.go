package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// TagRemoved TAG_REMOVEDイベントペイロード
type TagRemoved struct {
	Base
	TagID uuid.UUID `json:"tagId"`
	Tag   string    `json:"tag"`
}

func MakeTagRemoved(et time.Time, tag *model.Tag) *TagRemoved {
	return &TagRemoved{
		Base:  MakeBase(et),
		TagID: tag.ID,
		Tag:   tag.Name,
	}
}
