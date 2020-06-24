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
	"FileManager",
	"Imaging",
	"Notification",
	"RBAC",
	"ViewerManager",
	"WebRTCv3",
	"WS",
))
