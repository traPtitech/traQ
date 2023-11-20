package payload

import (
	"time"

	"github.com/gofrs/uuid"
)

// UserGroupDeleted USER_GROUP_DELETEDイベントペイロード
type UserGroupDeleted struct {
	Base
	GroupID uuid.UUID `json:"groupId"`
}

func MakeUserGroupDeleted(eventTime time.Time, groupID uuid.UUID) *UserGroupDeleted {
	return &UserGroupDeleted{
		Base:    MakeBase(eventTime),
		GroupID: groupID,
	}
}
