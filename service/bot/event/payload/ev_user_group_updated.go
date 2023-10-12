package payload

import (
	"time"

	"github.com/gofrs/uuid"
)

// UserGroupUpdated USER_GROUP_UPDATEDイベントペイロード
type UserGroupUpdated struct {
	Base
	GroupID uuid.UUID `json:"groupId"`
}

func MakeUserGroupUpdated(eventTime time.Time, groupID uuid.UUID) *UserGroupUpdated {
	return &UserGroupUpdated{
		Base:    MakeBase(eventTime),
		GroupID: groupID,
	}
}
