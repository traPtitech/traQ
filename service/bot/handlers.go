package bot

import (
	"github.com/leandro-lugaresi/hub"
	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/service/bot/handler"
	"time"
)

type eventHandler func(ctx handler.Context, datetime time.Time, event string, fields hub.Fields) error

var eventHandlerSet = map[string]eventHandler{
	intevent.BotJoined:           handler.BotJoined,
	intevent.BotLeft:             handler.BotLeft,
	intevent.BotPingRequest:      handler.BotPingRequest,
	intevent.MessageCreated:      handler.MessageCreated,
	intevent.MessageDeleted:      handler.MessageDeleted,
	intevent.MessageUpdated:      handler.MessageUpdated,
	intevent.UserCreated:         handler.UserCreated,
	intevent.ChannelCreated:      handler.ChannelCreated,
	intevent.ChannelTopicUpdated: handler.ChannelTopicUpdated,
	intevent.StampCreated:        handler.StampCreated,
	intevent.UserTagAdded:        handler.UserTagAdded,
	intevent.UserTagRemoved:      handler.UserTagRemoved,
}
