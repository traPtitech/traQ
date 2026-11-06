package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// UserGroupCreatedPayload UserGroupCreated のペイロード
type UserGroupCreatedPayload struct {
	GroupID uuid.UUID        `hub:"group_id"`
	Group   *model.UserGroup `hub:"group"`
}

func (UserGroupCreatedPayload) TopicName() string { return UserGroupCreated }

// UserGroupUpdatedPayload UserGroupUpdated のペイロード
type UserGroupUpdatedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
}

func (UserGroupUpdatedPayload) TopicName() string { return UserGroupUpdated }

// UserGroupDeletedPayload UserGroupDeleted のペイロード
type UserGroupDeletedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
}

func (UserGroupDeletedPayload) TopicName() string { return UserGroupDeleted }

// UserGroupMemberAddedPayload UserGroupMemberAdded のペイロード
type UserGroupMemberAddedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
	UserID  uuid.UUID `hub:"user_id"`
}

func (UserGroupMemberAddedPayload) TopicName() string { return UserGroupMemberAdded }

// UserGroupMemberUpdatedPayload UserGroupMemberUpdated のペイロード
type UserGroupMemberUpdatedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
	UserID  uuid.UUID `hub:"user_id"`
}

func (UserGroupMemberUpdatedPayload) TopicName() string { return UserGroupMemberUpdated }

// UserGroupMemberRemovedPayload UserGroupMemberRemoved のペイロード
type UserGroupMemberRemovedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
	UserID  uuid.UUID `hub:"user_id"`
}

func (UserGroupMemberRemovedPayload) TopicName() string { return UserGroupMemberRemoved }

// UserGroupAdminAddedPayload UserGroupAdminAdded のペイロード
type UserGroupAdminAddedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
	UserID  uuid.UUID `hub:"user_id"`
}

func (UserGroupAdminAddedPayload) TopicName() string { return UserGroupAdminAdded }

// UserGroupAdminRemovedPayload UserGroupAdminRemoved のペイロード
type UserGroupAdminRemovedPayload struct {
	GroupID uuid.UUID `hub:"group_id"`
	UserID  uuid.UUID `hub:"user_id"`
}

func (UserGroupAdminRemovedPayload) TopicName() string { return UserGroupAdminRemoved }

var (
	_ Payload = UserGroupCreatedPayload{}
	_ Payload = UserGroupUpdatedPayload{}
	_ Payload = UserGroupDeletedPayload{}
	_ Payload = UserGroupMemberAddedPayload{}
	_ Payload = UserGroupMemberUpdatedPayload{}
	_ Payload = UserGroupMemberRemovedPayload{}
	_ Payload = UserGroupAdminAddedPayload{}
	_ Payload = UserGroupAdminRemovedPayload{}
)
