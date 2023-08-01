package payload

import (
	"github.com/gofrs/uuid"
	"time"
)

// UserGroupAdminAdded USER_GROUP_ADMIN_ADDEDイベントペイロード
type UserGroupAdminAdded struct {
	Base
	GroupMember struct {
		GroupID uuid.UUID `json:"groupId"`
		UserID  uuid.UUID `json:"userId"`
	} `json:"groupMember"`
}

func MakeUserGroupAdminAdded(eventTime time.Time, groupID, userID uuid.UUID) *UserGroupAdminAdded {
	return &UserGroupAdminAdded{
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
