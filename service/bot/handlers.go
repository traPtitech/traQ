package bot

import (
	"github.com/leandro-lugaresi/hub"
	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/service/bot/handler"
)

type eventHandler func(ctx handler.Context, event string, fields hub.Fields)

var eventHandlerSet = map[string]eventHandler{
	intevent.BotJoined:           handler.BotJoined,
	intevent.BotLeft:             handler.BotLeft,
	intevent.BotPingRequest:      handler.BotPingRequest,
	intevent.MessageCreated:      handler.MessageCreated,
	intevent.UserCreated:         handler.UserCreated,
	intevent.ChannelCreated:      handler.ChannelCreated,
	intevent.ChannelTopicUpdated: handler.ChannelTopicUpdated,
	intevent.StampCreated:        handler.StampCreated,
	intevent.UserTagAdded:        handler.UserTagAdded,
	intevent.UserTagRemoved:      handler.UserTagRemoved,
}
