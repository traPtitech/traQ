package bot

import (
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/service/bot/handler"
)

type eventHandler func(ctx handler.Context, datetime time.Time, event string, fields hub.Fields) error

var eventHandlerSet = map[string]eventHandler{
	event.BotJoined:              handler.BotJoined,
	event.BotLeft:                handler.BotLeft,
	event.BotPingRequest:         handler.BotPingRequest,
	event.MessageCreated:         handler.MessageCreated,
	event.MessageDeleted:         handler.MessageDeleted,
	event.MessageUpdated:         handler.MessageUpdated,
	event.UserCreated:            handler.UserCreated,
	event.UserActivated:          handler.UserActivated,
	event.ChannelCreated:         handler.ChannelCreated,
	event.ChannelTopicUpdated:    handler.ChannelTopicUpdated,
	event.StampCreated:           handler.StampCreated,
	event.StampUpdated:           handler.StampUpdated,
	event.StampDeleted:           handler.StampDeleted,
	event.UserTagAdded:           handler.UserTagAdded,
	event.UserTagRemoved:         handler.UserTagRemoved,
	event.MessageStampsUpdated:   handler.MessageStampsUpdated,
	event.UserGroupCreated:       handler.UserGroupCreated,
	event.UserGroupUpdated:       handler.UserGroupUpdated,
	event.UserGroupDeleted:       handler.UserGroupDeleted,
	event.UserGroupMemberAdded:   handler.UserGroupMemberAdded,
	event.UserGroupMemberUpdated: handler.UserGroupMemberUpdated,
	event.UserGroupMemberRemoved: handler.UserGroupMemberRemoved,
	event.UserGroupAdminAdded:    handler.UserGroupAdminAdded,
	event.UserGroupAdminRemoved:  handler.UserGroupAdminRemoved,
}
