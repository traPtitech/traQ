package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupMemberAdded USER_GROUP_MEMBER_ADDEDイベントペイロード
type UserGroupMemberAdded struct {
	Base
	GroupMember `json:"groupMember"`
}

func MakeUserGroupMemberAdded(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupMemberAdded {
	return &UserGroupMemberAdded{
		Base:        MakeBase(eventTime),
		GroupMember: MakeGroupMember(groupID, userID),
	}
}
