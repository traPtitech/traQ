package v4

import (
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/router/v4/messages"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/ogp"
	"github.com/traPtitech/traQ/service/oidc"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	mutil "github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"

	botWS "github.com/traPtitech/traQ/service/bot/ws"
)

type Handlers struct {
	RBAC           rbac.RBAC
	Repo           repository.Repository
	WS             *ws.Streamer
	BotWS          *botWS.Streamer
	Hub            *hub.Hub
	Logger         *zap.Logger
	OC             *counter.OnlineCounter
	OGP            ogp.Service
	OIDC           *oidc.Service
	VM             *viewer.Manager
	WebRTC         *webrtcv3.Manager
	Imaging        imaging.Processor
	SessStore      session.Store
	SearchEngine   search.Engine
	ChannelManager channel.Manager
	MessageManager message.Manager
	FileManager    file.Manager
	Replacer       *mutil.Replacer
	Soundboard     qall.Soundboard
	QallRepo       qall.RoomStateManager
}

type Services struct {
	MessageService *messages.Service
}

func (h *Services) SetUp(e *echo.Group) {
	// 認証なしでConnectRPCサービスをマウント
	api := e.Group("/v4")
	{
		// パスを使ってルーティング
		api.Any(h.MessageService.Path+"*", echo.WrapHandler(h.MessageService.Handler))
	}
}
