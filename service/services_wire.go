// +build wireinject

package service

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(wire.FieldsOf(new(*Services),
	"BOT",
	"OnlineCounter",
	"UnreadMessageCounter",
	"MessageCounter",
	"ChannelCounter",
	"FCM",
	"HeartBeats",
	"Imaging",
	"Notification",
	"RBAC",
	"Search",
	"SSE",
	"ViewerManager",
	"WebRTCv3",
	"WS",
))
