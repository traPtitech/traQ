package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupAdminRemoved USER_GROUP_ADMIN_REMOVEDイベントペイロード
type UserGroupAdminRemoved struct {
	Base
	GroupMember struct {
		GroupID uuid.UUID `json:"groupId"`
		UserID  uuid.UUID `json:"userId"`
	} `json:"groupMember"`
}

func MakeUserGroupAdminRemoved(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupAdminRemoved {
	return &UserGroupAdminRemoved{
		Base: MakeBase(eventTime),
		GroupMember: struct {
			GroupID uuid.UUID `json:"groupId"`
			UserID  uuid.UUID `json:"userId"`
		}{
			GroupID: groupID,
			UserID:  userID,
		},
	}
}
