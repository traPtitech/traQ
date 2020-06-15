package service

import (
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
)

type Services struct {
	BOT                  bot.Service
	ChannelManager       channel.Manager
	OnlineCounter        *counter.OnlineCounter
	UnreadMessageCounter counter.UnreadMessageCounter
	MessageCounter       counter.MessageCounter
	ChannelCounter       counter.ChannelCounter
	FCM                  fcm.Client
	FileManager          file.Manager
	Imaging              imaging.Processor
	Notification         *notification.Service
	RBAC                 rbac.RBAC
	ViewerManager        *viewer.Manager
	WebRTCv3             *webrtcv3.Manager
	WS                   *ws.Streamer
}
