package v3

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

type tagResponse struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func formatTag(ut *model.UsersTag) tagResponse {
	return tagResponse{
		ID:        ut.Tag.ID,
		Tag:       ut.Tag.Name,
		IsLocked:  ut.IsLocked,
		CreatedAt: ut.CreatedAt,
		UpdatedAt: ut.UpdatedAt,
	}
}

func formatTags(uts []*model.UsersTag) []tagResponse {
	res := make([]tagResponse, len(uts))
	for i, ut := range uts {
		res[i] = formatTag(ut)
	}
	return res
}
