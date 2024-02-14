package v1

import (
	"encoding/gob"
	"net/http"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	mutil "github.com/traPtitech/traQ/utils/message"

	"github.com/labstack/echo/v4"
)

var json = jsonIter.ConfigFastest

func init() {
	gob.Register(uuid.UUID{})
}

// Handlers ハンドラ
type Handlers struct {
	RBAC           rbac.RBAC
	Repo           repository.Repository
	Hub            *hub.Hub
	Logger         *zap.Logger
	SessStore      session.Store
	ChannelManager channel.Manager
	MessageManager message.Manager
	FileManager    file.Manager
	Replacer       *mutil.Replacer
	EmojiCache     *EmojiCache
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	retrieve := middlewares.NewParamRetriever(h.Repo, h.ChannelManager, h.FileManager, h.MessageManager)
	blockBot := middlewares.BlockBot()

	requiresFileAccessPerm := middlewares.CheckFileAccessPerm(h.FileManager)

	gone := func(_ echo.Context) error {
		return herror.HTTPError(http.StatusGone, "This API has been deleted. Please migrate to v3 or newer API.")
	}

	api := e.Group("/1.0", middlewares.UserAuthenticate(h.Repo, h.SessStore))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", gone)
			apiUsers.POST("", gone)
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", gone)
				apiUsersMe.PATCH("", gone)
				apiUsersMe.PUT("/password", gone)
				apiUsersMe.GET("/qr-code", gone)
				apiUsersMe.GET("/icon", gone)
				apiUsersMe.PUT("/icon", gone)
				apiUsersMe.GET("/stamp-history", gone)
				apiUsersMe.GET("/groups", gone)
				apiUsersMe.GET("/notification", gone)
				apiUsersMeSessions := apiUsersMe.Group("/sessions")
				{
					apiUsersMeSessions.GET("", gone)
					apiUsersMeSessions.DELETE("", gone)
					apiUsersMeSessions.DELETE("/:referenceID", gone)
				}
				apiUsersMeStars := apiUsersMe.Group("/stars")
				{
					apiUsersMeStars.GET("", gone)
					apiUsersMeStarsCid := apiUsersMeStars.Group("/:channelID")
					{
						apiUsersMeStarsCid.PUT("", gone)
						apiUsersMeStarsCid.DELETE("", gone)
					}
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread")
				{
					apiUsersMeUnread.GET("/channels", gone)
					apiUsersMeUnread.DELETE("/channels/:channelID", gone)
				}
				apiUsersMeTokens := apiUsersMe.Group("/tokens")
				{
					apiUsersMeTokens.GET("", gone)
					apiUsersMeTokens.DELETE("/:tokenID", gone)
				}
			}
			apiUsersUID := apiUsers.Group("/:userID")
			{
				apiUsersUID.GET("", gone)
				apiUsersUID.PATCH("", gone)
				apiUsersUID.PUT("/status", gone)
				apiUsersUID.PUT("/password", gone)
				apiUsersUID.GET("/messages", gone)
				apiUsersUID.POST("/messages", gone)
				apiUsersUID.GET("/icon", gone)
				apiUsersUID.PUT("/icon", gone)
				apiUsersUID.GET("/notification", gone)
				apiUsersUID.GET("/groups", gone)
				apiUsersUIDTags := apiUsersUID.Group("/tags")
				{
					apiUsersUIDTags.GET("", gone)
					apiUsersUIDTags.POST("", gone)
					apiUsersUIDTagsTid := apiUsersUIDTags.Group("/:tagID")
					{
						apiUsersUIDTagsTid.PATCH("", gone)
						apiUsersUIDTagsTid.DELETE("", gone)
					}
				}
			}
		}
		apiHeartBeat := api.Group("/heartbeat")
		{
			apiHeartBeat.GET("", gone)
			apiHeartBeat.POST("", gone)
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", gone)
			apiChannels.POST("", gone)
			apiChannelsCid := apiChannels.Group("/:channelID")
			{
				apiChannelsCid.GET("", gone)
				apiChannelsCid.PATCH("", gone)
				apiChannelsCid.PUT("/parent", gone)
				apiChannelsCid.POST("/children", gone)
				apiChannelsCid.GET("/pins", gone)
				apiChannelsCid.GET("/events", gone)
				apiChannelsCid.GET("/stats", gone)
				apiChannelsCid.GET("/viewers", gone)
				apiChannelsCidTopic := apiChannelsCid.Group("/topic")
				{
					apiChannelsCidTopic.GET("", gone)
					apiChannelsCidTopic.PUT("", gone)
				}
				apiChannelsCidMessages := apiChannelsCid.Group("/messages")
				{
					apiChannelsCidMessages.GET("", gone)
					apiChannelsCidMessages.POST("", gone)
				}
				apiChannelsCidNotification := apiChannelsCid.Group("/notification")
				{
					apiChannelsCidNotification.GET("", gone)
					apiChannelsCidNotification.PUT("", gone)
				}
				apiChannelsCidBots := apiChannelsCid.Group("/bots")
				{
					apiChannelsCidBots.GET("", gone)
					apiChannelsCidBots.POST("", gone)
					apiChannelsCidBots.DELETE("/:botID", gone)
				}
				apiChannelsCidWebRTC := apiChannelsCid.Group("/webrtc")
				{
					apiChannelsCidWebRTC.GET("/state", gone)
				}
			}
		}
		apiNotification := api.Group("/notification")
		{
			apiNotification.GET("", gone)
			apiNotification.POST("/device", gone)
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("/reports", gone)
			apiMessagesMid := apiMessages.Group("/:messageID")
			{
				apiMessagesMid.GET("", gone)
				apiMessagesMid.PUT("", gone)
				apiMessagesMid.DELETE("", gone)
				apiMessagesMid.POST("/report", gone)
				apiMessagesMid.GET("/stamps", gone)
				apiMessagesMidStampsSid := apiMessagesMid.Group("/stamps/:stampID")
				{
					apiMessagesMidStampsSid.POST("", gone)
					apiMessagesMidStampsSid.DELETE("", gone)
				}
			}
		}
		apiTags := api.Group("/tags")
		{
			apiTagsTid := apiTags.Group("/:tagID")
			{
				apiTagsTid.GET("", gone)
			}
		}
		apiFiles := api.Group("/files")
		{
			apiFiles.POST("", gone)
			apiFilesFid := apiFiles.Group("/:fileID", retrieve.FileID(), requiresFileAccessPerm)
			{
				apiFilesFid.GET("", h.GetFileByID, requires(permission.DownloadFile))
				apiFilesFid.GET("/meta", h.GetMetaDataByFileID, requires(permission.DownloadFile))
				apiFilesFid.GET("/thumbnail", h.GetThumbnailByID, requires(permission.DownloadFile))
			}
		}
		apiPins := api.Group("/pins")
		{
			apiPins.POST("", gone)
			apiPinsPid := apiPins.Group("/:pinID")
			{
				apiPinsPid.GET("", gone)
				apiPinsPid.DELETE("", gone)
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", gone)
			apiStamps.POST("", gone)
			apiStampsSid := apiStamps.Group("/:stampID")
			{
				apiStampsSid.GET("", gone)
				apiStampsSid.PATCH("", gone)
				apiStampsSid.DELETE("", gone)
			}
		}
		apiWebhooks := api.Group("/webhooks", blockBot)
		{
			apiWebhooks.GET("", gone)
			apiWebhooks.POST("", gone)
			apiWebhooksWid := apiWebhooks.Group("/:webhookID")
			{
				apiWebhooksWid.GET("", gone)
				apiWebhooksWid.PATCH("", gone)
				apiWebhooksWid.DELETE("", gone)
				apiWebhooksWid.GET("/icon", gone)
				apiWebhooksWid.PUT("/icon", gone)
				apiWebhooksWid.GET("/messages", gone)
			}
		}
		apiGroups := api.Group("/groups")
		{
			apiGroups.GET("", gone)
			apiGroups.POST("", gone)
			apiGroupsGid := apiGroups.Group("/:groupID")
			{
				apiGroupsGid.GET("", gone)
				apiGroupsGid.PATCH("", gone)
				apiGroupsGid.DELETE("", gone)
				apiGroupsGidMembers := apiGroupsGid.Group("/members")
				{
					apiGroupsGidMembers.GET("", gone)
					apiGroupsGidMembers.POST("", gone)
					apiGroupsGidMembers.DELETE("/:userID", gone)
				}
			}
		}
		apiClients := api.Group("/clients")
		{
			apiClients.GET("", gone)
			apiClients.POST("", gone)
			apiClientCid := apiClients.Group("/:clientID")
			{
				apiClientCid.GET("", gone)
				apiClientCid.PATCH("", gone)
				apiClientCid.DELETE("", gone)
				apiClientCid.GET("/detail", gone)
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", gone)
			apiBots.POST("", gone)
			apiBotsBid := apiBots.Group("/:botID")
			{
				apiBotsBid.GET("", gone)
				apiBotsBid.PATCH("", gone)
				apiBotsBid.DELETE("", gone)
				apiBotsBid.GET("/detail", gone)
				apiBotsBid.PUT("/events", gone)
				apiBotsBid.GET(`/events/logs`, gone)
				apiBotsBid.GET("/icon", gone)
				apiBotsBid.PUT("/icon", gone)
				apiBotsBid.PUT("/state", gone)
				apiBotsBid.POST("/reissue", gone)
				apiBotsBid.GET("/channels", gone)
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/latest-messages", gone)
		}
		apiWebRTC := api.Group("/webrtc")
		{
			apiWebRTC.GET("/state", gone)
			apiWebRTC.PUT("/state", gone)
		}
		api.POST("/skyway/authenticate", gone)
	}

	apiNoAuth := e.Group("/1.0")
	{
		apiNoAuth.POST("/login", gone)
		apiNoAuth.POST("/logout", gone)
		apiPublic := apiNoAuth.Group("/public")
		{
			apiPublic.GET("/icon/:username", h.GetPublicUserIcon)
			apiPublic.GET("/emoji.json", h.GetPublicEmojiJSON)
			apiPublic.GET("/emoji.css", h.GetPublicEmojiCSS)
			apiPublic.GET("/emoji/:stampID", h.GetPublicEmojiImage, retrieve.StampID(false))
		}
		apiNoAuth.POST("/webhooks/:webhookID", gone)
		apiNoAuth.POST("/webhooks/:webhookID/github", gone)
	}

	go h.stampEventSubscriber(h.Hub.Subscribe(10, event.StampCreated, event.StampUpdated, event.StampDeleted))
}

func (h *Handlers) stampEventSubscriber(sub hub.Subscription) {
	for range sub.Receiver {
		h.EmojiCache.Purge()
	}
}

func getStampFromContext(c echo.Context) *model.Stamp {
	return c.Get(consts.KeyParamStamp).(*model.Stamp)
}

func getFileFromContext(c echo.Context) model.File {
	return c.Get(consts.KeyParamFile).(model.File)
}

// L ロガーを返します
func (h *Handlers) L(c echo.Context) *zap.Logger {
	return h.Logger.With(zap.String("requestId", extension.GetRequestID(c)))
}
