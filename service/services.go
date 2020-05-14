package service

import (
	"github.com/traPtitech/traQ/service/bot"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/heartbeat"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
)

type Services struct {
	BOT                  *bot.Processor
	OnlineCounter        *counter.OnlineCounter
	UnreadMessageCounter counter.UnreadMessageCounter
	MessageCounter       counter.MessageCounter
	ChannelCounter       counter.ChannelCounter
	FCM                  *fcm.Client
	HeartBeats           *heartbeat.Manager
	Imaging              imaging.Processor
	Notification         *notification.Service
	RBAC                 rbac.RBAC
	SSE                  *sse.Streamer
	ViewerManager        *viewer.Manager
	WebRTCv3             *webrtcv3.Manager
	WS                   *ws.Streamer
}
