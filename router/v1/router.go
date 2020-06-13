package v1

import (
	"bytes"
	"encoding/gob"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	imaging2 "github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

var json = jsoniter.ConfigFastest

func init() {
	gob.Register(uuid.UUID{})
}

// Handlers ハンドラ
type Handlers struct {
	RBAC           rbac.RBAC
	Repo           repository.Repository
	SSE            *sse.Streamer
	Hub            *hub.Hub
	Logger         *zap.Logger
	OC             *counter.OnlineCounter
	VM             *viewer.Manager
	Imaging        imaging2.Processor
	SessStore      session.Store
	ChannelManager channel.Manager
	Replacer       *message.Replacer

	emojiJSONCache     bytes.Buffer `wire:"-"`
	emojiJSONTime      time.Time    `wire:"-"`
	emojiJSONCacheLock sync.RWMutex `wire:"-"`
	emojiCSSCache      bytes.Buffer `wire:"-"`
	emojiCSSTime       time.Time    `wire:"-"`
	emojiCSSCacheLock  sync.RWMutex `wire:"-"`
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := middlewares.RequestBodyLengthLimit
	retrieve := middlewares.NewParamRetriever(h.Repo, h.ChannelManager)
	blockBot := middlewares.BlockBot(h.Repo)
	nologin := middlewares.NoLogin(h.SessStore)

	requiresBotAccessPerm := middlewares.CheckBotAccessPerm(h.RBAC, h.Repo)
	requiresWebhookAccessPerm := middlewares.CheckWebhookAccessPerm(h.RBAC, h.Repo)
	requiresFileAccessPerm := middlewares.CheckFileAccessPerm(h.RBAC, h.Repo)
	requiresClientAccessPerm := middlewares.CheckClientAccessPerm(h.RBAC, h.Repo)
	requiresMessageAccessPerm := middlewares.CheckMessageAccessPerm(h.RBAC, h.ChannelManager)
	requiresChannelAccessPerm := middlewares.CheckChannelAccessPerm(h.RBAC, h.ChannelManager)

	gone := func(c echo.Context) error { return herror.HTTPError(http.StatusGone, "this api has been deleted") }

	api := e.Group("/1.0", middlewares.UserAuthenticate(h.Repo, h.SessStore))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", h.GetUsers, requires(permission.GetUser))
			apiUsers.POST("", h.PostUsers, requires(permission.RegisterUser))
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", h.GetMe, requires(permission.GetMe))
				apiUsersMe.PATCH("", h.PatchMe, requires(permission.EditMe))
				apiUsersMe.PUT("/password", h.PutPassword, requires(permission.ChangeMyPassword), blockBot)
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.GetUserQRCode), blockBot)
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.PutMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/groups", h.GetMyBelongingGroup, requires(permission.GetUserGroup))
				apiUsersMe.GET("/notification", h.GetMyNotificationChannels, requires(permission.GetChannelSubscription), blockBot)
				apiUsersMeSessions := apiUsersMe.Group("/sessions", blockBot)
				{
					apiUsersMeSessions.GET("", h.GetMySessions, requires(permission.GetMySessions))
					apiUsersMeSessions.DELETE("", h.DeleteAllMySessions, requires(permission.DeleteMySessions))
					apiUsersMeSessions.DELETE("/:referenceID", h.DeleteMySession, requires(permission.DeleteMySessions))
				}
				apiUsersMeStars := apiUsersMe.Group("/stars", blockBot)
				{
					apiUsersMeStars.GET("", h.GetStars, requires(permission.GetChannelStar))
					apiUsersMeStarsCid := apiUsersMeStars.Group("/:channelID", retrieve.ChannelID(), requiresChannelAccessPerm)
					{
						apiUsersMeStarsCid.PUT("", h.PutStars, requires(permission.EditChannelStar))
						apiUsersMeStarsCid.DELETE("", h.DeleteStars, requires(permission.EditChannelStar))
					}
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread", blockBot)
				{
					apiUsersMeUnread.GET("/channels", h.GetUnreadChannels, requires(permission.GetUnread))
					apiUsersMeUnread.DELETE("/channels/:channelID", h.DeleteUnread, requires(permission.DeleteUnread))
				}
				apiUsersMeTokens := apiUsersMe.Group("/tokens", blockBot)
				{
					apiUsersMeTokens.GET("", h.GetMyTokens, requires(permission.GetMyTokens))
					apiUsersMeTokens.DELETE("/:tokenID", h.DeleteMyToken, requires(permission.RevokeMyToken))
				}
			}
			apiUsersUID := apiUsers.Group("/:userID", retrieve.UserID(false))
			{
				apiUsersUID.GET("", h.GetUserByID, requires(permission.GetUser))
				apiUsersUID.PATCH("", h.PatchUserByID, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/status", h.PutUserStatus, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/password", h.PutUserPassword, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/messages", h.GetDirectMessages, requires(permission.GetMessage))
				apiUsersUID.POST("/messages", h.PostDirectMessage, bodyLimit(100), requires(permission.PostMessage))
				apiUsersUID.GET("/icon", h.GetUserIcon, requires(permission.DownloadFile))
				apiUsersUID.PUT("/icon", h.PutUserIcon, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/notification", h.GetNotificationChannels, requires(permission.GetChannelSubscription))
				apiUsersUID.GET("/groups", h.GetUserBelongingGroup, requires(permission.GetUserGroup))
				apiUsersUIDTags := apiUsersUID.Group("/tags")
				{
					apiUsersUIDTags.GET("", h.GetUserTags, requires(permission.GetUserTag))
					apiUsersUIDTags.POST("", h.PostUserTag, requires(permission.EditUserTag))
					apiUsersUIDTagsTid := apiUsersUIDTags.Group("/:tagID")
					{
						apiUsersUIDTagsTid.PATCH("", h.PatchUserTag, requires(permission.EditUserTag))
						apiUsersUIDTagsTid.DELETE("", h.DeleteUserTag, requires(permission.EditUserTag))
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
			apiChannels.POST("", h.PostChannels, requires(permission.CreateChannel))
			apiChannelsCid := apiChannels.Group("/:channelID", retrieve.ChannelID(), requiresChannelAccessPerm)
			{
				apiChannelsCid.GET("", h.GetChannelByChannelID, requires(permission.GetChannel))
				apiChannelsCid.PATCH("", gone)
				apiChannelsCid.PUT("/parent", gone)
				apiChannelsCid.POST("/children", h.PostChannelChildren, requires(permission.CreateChannel))
				apiChannelsCid.GET("/pins", gone)
				apiChannelsCid.GET("/events", h.GetChannelEvents, requires(permission.GetChannel))
				apiChannelsCid.GET("/stats", h.GetChannelStats, requires(permission.GetChannel))
				apiChannelsCid.GET("/viewers", h.GetChannelViewers, requires(permission.GetChannel))
				apiChannelsCidTopic := apiChannelsCid.Group("/topic")
				{
					apiChannelsCidTopic.GET("", h.GetTopic, requires(permission.GetChannel))
					apiChannelsCidTopic.PUT("", h.PutTopic, requires(permission.EditChannelTopic))
				}
				apiChannelsCidMessages := apiChannelsCid.Group("/messages")
				{
					apiChannelsCidMessages.GET("", h.GetMessagesByChannelID, requires(permission.GetMessage))
					apiChannelsCidMessages.POST("", h.PostMessage, bodyLimit(100), requires(permission.PostMessage))
				}
				apiChannelsCidNotification := apiChannelsCid.Group("/notification")
				{
					apiChannelsCidNotification.GET("", h.GetChannelSubscribers, requires(permission.GetChannelSubscription))
					apiChannelsCidNotification.PUT("", h.PutChannelSubscribers, requires(permission.EditChannelSubscription))
				}
				apiChannelsCidBots := apiChannelsCid.Group("/bots", blockBot)
				{
					apiChannelsCidBots.GET("", h.GetChannelBots, requires(permission.GetBot))
					apiChannelsCidBots.POST("", h.PostChannelBots, requires(permission.BotActionJoinChannel))
					apiChannelsCidBots.DELETE("/:botID", h.DeleteChannelBot, requires(permission.BotActionLeaveChannel), retrieve.BotID())
				}
				apiChannelsCidWebRTC := apiChannelsCid.Group("/webrtc")
				{
					apiChannelsCidWebRTC.GET("/state", gone)
				}
			}
		}
		apiNotification := api.Group("/notification", blockBot)
		{
			apiNotification.GET("", echo.WrapHandler(h.SSE), requires(permission.ConnectNotificationStream))
			apiNotification.POST("/device", h.PostDeviceToken, requires(permission.RegisterFCMDevice))
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("/reports", h.GetMessageReports, requires(permission.GetMessageReports), blockBot)
			apiMessagesMid := apiMessages.Group("/:messageID", retrieve.MessageID(), requiresMessageAccessPerm)
			{
				apiMessagesMid.GET("", h.GetMessageByID, requires(permission.GetMessage))
				apiMessagesMid.PUT("", h.PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMid.DELETE("", h.DeleteMessageByID, requires(permission.DeleteMessage))
				apiMessagesMid.POST("/report", h.PostMessageReport, requires(permission.ReportMessage), blockBot)
				apiMessagesMid.GET("/stamps", h.GetMessageStamps, requires(permission.GetMessage))
				apiMessagesMidStampsSid := apiMessagesMid.Group("/stamps/:stampID", retrieve.StampID(true))
				{
					apiMessagesMidStampsSid.POST("", h.PostMessageStamp, requires(permission.AddMessageStamp))
					apiMessagesMidStampsSid.DELETE("", h.DeleteMessageStamp, requires(permission.RemoveMessageStamp))
				}
			}
		}
		apiTags := api.Group("/tags")
		{
			apiTagsTid := apiTags.Group("/:tagID")
			{
				apiTagsTid.GET("", h.GetUsersByTagID, requires(permission.GetUserTag))
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
			apiStamps.GET("", h.GetStamps, requires(permission.GetStamp))
			apiStamps.POST("", h.PostStamp, requires(permission.CreateStamp))
			apiStampsSid := apiStamps.Group("/:stampID", retrieve.StampID(false))
			{
				apiStampsSid.GET("", h.GetStamp, requires(permission.GetStamp))
				apiStampsSid.PATCH("", h.PatchStamp, requires(permission.EditStamp))
				apiStampsSid.DELETE("", h.DeleteStamp, requires(permission.DeleteStamp))
			}
		}
		apiWebhooks := api.Group("/webhooks", blockBot)
		{
			apiWebhooks.GET("", h.GetWebhooks, requires(permission.GetWebhook))
			apiWebhooks.POST("", h.PostWebhooks, requires(permission.CreateWebhook))
			apiWebhooksWid := apiWebhooks.Group("/:webhookID", retrieve.WebhookID(), requiresWebhookAccessPerm)
			{
				apiWebhooksWid.GET("", h.GetWebhook, requires(permission.GetWebhook))
				apiWebhooksWid.PATCH("", h.PatchWebhook, requires(permission.EditWebhook))
				apiWebhooksWid.DELETE("", h.DeleteWebhook, requires(permission.DeleteWebhook))
				apiWebhooksWid.GET("/icon", h.GetWebhookIcon, requires(permission.GetWebhook))
				apiWebhooksWid.PUT("/icon", h.PutWebhookIcon, requires(permission.EditWebhook))
				apiWebhooksWid.GET("/messages", h.GetWebhookMessages, requires(permission.GetWebhook))
			}
		}
		apiGroups := api.Group("/groups")
		{
			apiGroups.GET("", h.GetUserGroups, requires(permission.GetUserGroup))
			apiGroups.POST("", h.PostUserGroups, requires(permission.CreateUserGroup))
			apiGroupsGid := apiGroups.Group("/:groupID", retrieve.GroupID())
			{
				apiGroupsGid.GET("", h.GetUserGroup, requires(permission.GetUserGroup))
				apiGroupsGid.PATCH("", h.PatchUserGroup, requires(permission.EditUserGroup))
				apiGroupsGid.DELETE("", h.DeleteUserGroup, requires(permission.DeleteUserGroup))
				apiGroupsGidMembers := apiGroupsGid.Group("/members")
				{
					apiGroupsGidMembers.GET("", h.GetUserGroupMembers, requires(permission.GetUserGroup))
					apiGroupsGidMembers.POST("", h.PostUserGroupMembers, requires(permission.EditUserGroup))
					apiGroupsGidMembers.DELETE("/:userID", h.DeleteUserGroupMembers, requires(permission.EditUserGroup))
				}
			}
		}
		apiClients := api.Group("/clients", blockBot)
		{
			apiClients.GET("", h.GetClients, requires(permission.GetClients))
			apiClients.POST("", h.PostClients, requires(permission.CreateClient))
			apiClientCid := apiClients.Group("/:clientID", retrieve.ClientID())
			{
				apiClientCid.GET("", h.GetClient, requires(permission.GetClients))
				apiClientCid.PATCH("", h.PatchClient, requires(permission.EditMyClient), requiresClientAccessPerm)
				apiClientCid.DELETE("", h.DeleteClient, requires(permission.DeleteMyClient), requiresClientAccessPerm)
				apiClientCid.GET("/detail", h.GetClientDetail, requires(permission.GetClients), requiresClientAccessPerm)
			}
		}
		apiBots := api.Group("/bots", blockBot)
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.PostBots, requires(permission.CreateBot))
			apiBotsBid := apiBots.Group("/:botID", retrieve.BotID())
			{
				apiBotsBid.GET("", h.GetBot, requires(permission.GetBot))
				apiBotsBid.PATCH("", h.PatchBot, requires(permission.EditBot), requiresBotAccessPerm)
				apiBotsBid.DELETE("", h.DeleteBot, requires(permission.DeleteBot), requiresBotAccessPerm)
				apiBotsBid.GET("/detail", h.GetBotDetail, requires(permission.GetBot), requiresBotAccessPerm)
				apiBotsBid.PUT("/events", h.PutBotEvents, requires(permission.EditBot), requiresBotAccessPerm)
				apiBotsBid.GET(`/events/logs`, h.GetBotEventLogs, requires(permission.GetBot), requiresBotAccessPerm)
				apiBotsBid.GET("/icon", h.GetBotIcon, requires(permission.GetBot))
				apiBotsBid.PUT("/icon", h.PutBotIcon, requires(permission.EditBot), requiresBotAccessPerm)
				apiBotsBid.PUT("/state", h.PutBotState, requires(permission.EditBot), requiresBotAccessPerm)
				apiBotsBid.POST("/reissue", h.PostBotReissueTokens, requires(permission.EditBot), requiresBotAccessPerm)
				apiBotsBid.GET("/channels", h.GetBotJoinChannels, requires(permission.GetBot), requiresBotAccessPerm)
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/latest-messages", h.GetActivityLatestMessages, requires(permission.GetMessage))
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
		apiNoAuth.POST("/login", h.PostLogin, nologin)
		apiNoAuth.POST("/logout", h.PostLogout)
		apiPublic := apiNoAuth.Group("/public")
		{
			apiPublic.GET("/icon/:username", h.GetPublicUserIcon)
			apiPublic.GET("/emoji.json", h.GetPublicEmojiJSON)
			apiPublic.GET("/emoji.css", h.GetPublicEmojiCSS)
			apiPublic.GET("/emoji/:stampID", h.GetPublicEmojiImage, retrieve.StampID(false))
		}
		apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook, retrieve.WebhookID())
		apiNoAuth.POST("/webhooks/:webhookID/github", gone)
	}

	go h.stampEventSubscriber(h.Hub.Subscribe(10, event.StampCreated, event.StampUpdated, event.StampDeleted))
}

func (h *Handlers) stampEventSubscriber(sub hub.Subscription) {
	for range sub.Receiver {
		h.emojiJSONCacheLock.Lock()
		h.emojiJSONCache.Reset()
		h.emojiJSONCacheLock.Unlock()

		h.emojiCSSCacheLock.Lock()
		h.emojiCSSCache.Reset()
		h.emojiCSSCacheLock.Unlock()
	}
}

func bindAndValidate(c echo.Context, i interface{}) error {
	return extension.BindAndValidate(c, i)
}

func getRequestUser(c echo.Context) model.UserInfo {
	return c.Get(consts.KeyUser).(model.UserInfo)
}

func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).GetID()
}

func getRequestParamAsUUID(c echo.Context, name string) uuid.UUID {
	return extension.GetRequestParamAsUUID(c, name)
}

func getGroupFromContext(c echo.Context) *model.UserGroup {
	return c.Get(consts.KeyParamGroup).(*model.UserGroup)
}

func getStampFromContext(c echo.Context) *model.Stamp {
	return c.Get(consts.KeyParamStamp).(*model.Stamp)
}

func getMessageFromContext(c echo.Context) *model.Message {
	return c.Get(consts.KeyParamMessage).(*model.Message)
}

func getChannelFromContext(c echo.Context) *model.Channel {
	return c.Get(consts.KeyParamChannel).(*model.Channel)
}

func getUserFromContext(c echo.Context) model.UserInfo {
	return c.Get(consts.KeyParamUser).(model.UserInfo)
}

func getWebhookFromContext(c echo.Context) model.Webhook {
	return c.Get(consts.KeyParamWebhook).(model.Webhook)
}

func getBotFromContext(c echo.Context) *model.Bot {
	return c.Get(consts.KeyParamBot).(*model.Bot)
}

func getFileFromContext(c echo.Context) model.FileMeta {
	return c.Get(consts.KeyParamFile).(model.FileMeta)
}

func getClientFromContext(c echo.Context) *model.OAuth2Client {
	return c.Get(consts.KeyParamClient).(*model.OAuth2Client)
}

// L ロガーを返します
func (h *Handlers) L(c echo.Context) *zap.Logger {
	return h.Logger.With(zap.String("requestId", extension.GetRequestID(c)))
}
