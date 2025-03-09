package service

import (
	"github.com/traPtitech/traQ/service/bot"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/exevent"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/notification"
	"github.com/traPtitech/traQ/service/ogp"
	"github.com/traPtitech/traQ/service/oidc"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/search"
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
	StampThrottler       *exevent.StampThrottler
	FCM                  fcm.Client
	FileManager          file.Manager
	Imaging              imaging.Processor
	MessageManager       message.Manager
	Notification         *notification.Service
	OGP                  ogp.Service
	OIDC                 *oidc.Service
	RBAC                 rbac.RBAC
	Search               search.Engine
	ViewerManager        *viewer.Manager
	WebRTCv3             *webrtcv3.Manager
	WS                   *ws.Streamer
	BotWS                *botWS.Streamer
	QallRoomStateManager qall.RoomStateManager
	QallSoundBoard       qall.Soundboard
}
