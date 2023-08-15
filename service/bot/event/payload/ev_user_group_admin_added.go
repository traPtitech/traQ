package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupAdminAdded USER_GROUP_ADMIN_ADDEDイベントペイロード
type UserGroupAdminAdded struct {
	Base
	GroupMember `json:"groupMember"`
}

func MakeUserGroupAdminAdded(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupAdminAdded {
	return &UserGroupAdminAdded{
		Base:        MakeBase(eventTime),
		GroupMember: MakeGroupMember(groupID, userID),
	}
}
