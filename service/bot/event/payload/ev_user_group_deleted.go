package payload

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// UserGroupDeleted USER_GROUP_DELETEDイベントペイロード
type UserGroupDeleted struct {
	Base
	GroupID uuid.UUID `json:"groupId"`
}

func MakeUserGroupDeleted(eventTime time.Time, group model.UserGroup) *UserGroupDeleted {
	return &UserGroupDeleted{
		Base:    MakeBase(eventTime),
		GroupID: group.ID,
	}
}
