//go:build wireinject
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
	"UserCounter",
	"ChannelCounter",
	"StampThrottler",
	"FCM",
	"FileManager",
	"Imaging",
	"MessageManager",
	"Notification",
	"OGP",
	"OIDC",
	"RBAC",
	"Search",
	"ViewerManager",
	"WebRTCv3",
	"WS",
	"BotWS",
	"QallRoomStateManager",
	"QallSoundBoard",
))
