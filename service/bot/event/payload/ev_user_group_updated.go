package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// UserGroupUpdated USER_GROUP_UPDATEDイベントペイロード
type UserGroupUpdated struct {
	Base
	Group model.UserGroup `json:"group"`
}

func MakeUserGroupUpdated(eventTime time.Time, group model.UserGroup) *UserGroupUpdated {
	return &UserGroupUpdated{
		Base:  MakeBase(eventTime),
		Group: group,
	}
}
