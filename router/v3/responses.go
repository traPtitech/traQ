package v3

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

type UserTag struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func formatUserTags(uts []*model.UsersTag) []UserTag {
	res := make([]UserTag, len(uts))
	for i, ut := range uts {
		res[i] = UserTag{
			ID:        ut.Tag.ID,
			Tag:       ut.Tag.Name,
			IsLocked:  ut.IsLocked,
			CreatedAt: ut.CreatedAt,
			UpdatedAt: ut.UpdatedAt,
		}
	}
	return res
}

type UserDetail struct {
	ID          uuid.UUID   `json:"id"`
	State       int         `json:"state"`
	Bot         bool        `json:"bot"`
	IconFileID  uuid.UUID   `json:"iconFileId"`
	DisplayName string      `json:"displayName"`
	Name        string      `json:"name"`
	TwitterID   string      `json:"twitterId"`
	LastOnline  *time.Time  `json:"lastOnline"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Tags        []UserTag   `json:"tags"`
	Groups      []uuid.UUID `json:"groups"`
	Bio         string      `json:"bio"`
}

func formatUserDetail(user *model.User, uts []*model.UsersTag, g []uuid.UUID) *UserDetail {
	u := &UserDetail{
		ID:          user.ID,
		State:       user.Status.Int(),
		Bot:         user.Bot,
		IconFileID:  user.Icon,
		DisplayName: user.DisplayName,
		Name:        user.Name,
		TwitterID:   user.TwitterID,
		LastOnline:  user.LastOnline.Ptr(),
		UpdatedAt:   user.UpdatedAt,
		Tags:        formatUserTags(uts),
		Groups:      g,
		Bio:         "", // TODO
	}

	if len(u.DisplayName) == 0 {
		u.DisplayName = u.Name
	}
	return u
}
