package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupMemberUpdated USER_GROUP_MEMBER_UPDATEDイベントペイロード
type UserGroupMemberUpdated struct {
	Base
	GroupMember `json:"groupMember"`
}

func MakeUserGroupMemberUpdated(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupMemberUpdated {
	return &UserGroupMemberUpdated{
		Base:        MakeBase(eventTime),
		GroupMember: MakeGroupMember(groupID, userID),
	}
}
