// +build wireinject

package service

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(wire.FieldsOf(new(*Services),
	"BOT",
	"ChannelManager",
	"OnlineCounter",
	"UnreadMessageCounter",
	"MessageCounter",
	"ChannelCounter",
	"FCM",
	"Imaging",
	"Notification",
	"RBAC",
	"SSE",
	"ViewerManager",
	"WebRTCv3",
	"WS",
))
