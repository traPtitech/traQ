package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupMemberRemoved USER_GROUP_MEMBER_REMOVEDイベントペイロード
type UserGroupMemberRemoved struct {
	Base
	GroupMember struct {
		GroupID uuid.UUID `json:"groupId"`
		UserID  uuid.UUID `json:"userId"`
	} `json:"groupMember"`
}

func MakeUserGroupMemberRemoved(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupMemberRemoved {
	return &UserGroupMemberRemoved{
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
