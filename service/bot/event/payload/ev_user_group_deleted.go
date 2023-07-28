package payload

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// UserGroupDeleted USER_GROUP_DELETEDイベントペイロード
type UserGroupDeleted struct {
	Base
	Group struct {
		ID uuid.UUID `json:"id"`
	} `json:"group"`
}

func MakeUserGroupDeleted(eventTime time.Time, group model.UserGroup) *UserGroupDeleted {
	return &UserGroupDeleted{
		Base: MakeBase(eventTime),
		Group: struct {
			ID uuid.UUID `json:"id"`
		}{ID: group.ID},
	}
}
