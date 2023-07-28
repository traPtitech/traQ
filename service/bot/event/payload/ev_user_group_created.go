package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// UserGroupCreated USER_GROUP_CREATEDイベントペイロード
type UserGroupCreated struct {
	Base
	Group model.UserGroup `json:"group"`
}

func MakeUserGroupCreated(eventTime time.Time, group model.UserGroup) *UserGroupCreated {
	return &UserGroupCreated{
		Base:  MakeBase(eventTime),
		Group: group,
	}
}
