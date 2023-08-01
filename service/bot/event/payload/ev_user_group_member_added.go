package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupMemberAdded USER_GROUP_MEMBER_ADDEDイベントペイロード
type UserGroupMemberAdded struct {
	Base
	GroupMember struct {
		GroupID uuid.UUID `json:"groupId"`
		UserID  uuid.UUID `json:"userId"`
	} `json:"groupMember"`
}

func MakeUserGroupMemberAdded(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupMemberAdded {
	return &UserGroupMemberAdded{
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
