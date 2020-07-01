package payload

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
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
