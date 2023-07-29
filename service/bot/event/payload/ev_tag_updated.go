package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// TagUpdated TAG_UPDATEDイベントペイロード
type TagUpdated struct {
	Base
	TagID uuid.UUID `json:"tagId"`
	Tag   string    `json:"tag"`
}

func MakeTagUpdated(et time.Time, tag *model.Tag) *TagUpdated {
	return &TagUpdated{
		Base:  MakeBase(et),
		TagID: tag.ID,
		Tag:   tag.Name,
	}
}
