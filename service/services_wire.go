// +build wireinject

package service

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(wire.FieldsOf(new(*Services),
	"BOT",
	"OnlineCounter",
	"FCM",
	"HeartBeats",
	"Imaging",
	"SSE",
	"ViewerManager",
	"WebRTCv3",
	"WS",
	"Notification",
))
