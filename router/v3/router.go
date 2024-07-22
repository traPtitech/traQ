package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/session"
	botWS "github.com/traPtitech/traQ/service/bot/ws"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/ogp"
	"github.com/traPtitech/traQ/service/oidc"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/webrtcv3"
	"github.com/traPtitech/traQ/service/ws"
	mutil "github.com/traPtitech/traQ/utils/message"
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
	Config
}

type Config struct {
	Version  string
	Revision string

	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string

	// AllowSignUp ユーザーが自分自身で登録できるかどうか
	AllowSignUp bool

	// EnabledExternalAccountLink リンク可能な外部認証アカウントのプロバイダ
	EnabledExternalAccountProviders map[string]bool
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := middlewares.RequestBodyLengthLimit
	retrieve := middlewares.NewParamRetriever(h.Repo, h.ChannelManager, h.FileManager, h.MessageManager)
	blockBot := middlewares.BlockBot()
	blockNonBot := middlewares.BlockNonBot()
	noLogin := middlewares.NoLogin(h.SessStore, h.Repo)

	requiresBotAccessPerm := middlewares.CheckBotAccessPerm(h.RBAC)
	requiresWebhookAccessPerm := middlewares.CheckWebhookAccessPerm(h.RBAC)
	requiresFileAccessPerm := middlewares.CheckFileAccessPerm(h.FileManager)
	requiresClientAccessPerm := middlewares.CheckClientAccessPerm(h.RBAC)
	requiresMessageAccessPerm := middlewares.CheckMessageAccessPerm(h.ChannelManager)
	requiresChannelAccessPerm := middlewares.CheckChannelAccessPerm(h.ChannelManager)
	requiresGroupAdminPerm := middlewares.CheckUserGroupAdminPerm(h.RBAC)
	requiresClipFolderAccessPerm := middlewares.CheckClipFolderAccessPerm()
	requiresDeleteStampPerm := middlewares.CheckDeleteStampPerm(h.RBAC)

	api := e.Group("/v3", middlewares.UserAuthenticate(h.Repo, h.SessStore))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", h.GetUsers, requires(permission.GetUser))
			if !h.Config.AllowSignUp {
				apiUsers.POST("", h.CreateUser, requires(permission.RegisterUser))
			}
			apiUsersUID := apiUsers.Group("/:userID", retrieve.UserID(false))
			{
				apiUsersUID.GET("", h.GetUser, requires(permission.GetUser))
				apiUsersUID.PATCH("", h.EditUser, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/dm-channel", h.GetUserDMChannel, requires(permission.GetChannel))
				apiUsersUID.GET("/messages", h.GetDirectMessages, requires(permission.GetMessage))
				apiUsersUID.GET("/stats", h.GetUserStats, requires(permission.GetUser))
				apiUsersUID.POST("/messages", h.PostDirectMessage, bodyLimit(100), requires(permission.PostMessage))
				apiUsersUID.GET("/icon", h.GetUserIcon, requires(permission.DownloadFile))
				apiUsersUID.PUT("/icon", h.ChangeUserIcon, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/password", h.ChangeUserPassword, requires(permission.EditOtherUsers))
				apiUsersUIDTags := apiUsersUID.Group("/tags")
				{
					apiUsersUIDTags.GET("", h.GetUserTags, requires(permission.GetUserTag))
					apiUsersUIDTags.POST("", h.AddUserTag, requires(permission.EditUserTag))
					apiUsersUIDTagsTID := apiUsersUIDTags.Group("/:tagID")
					{
						apiUsersUIDTagsTID.PATCH("", h.EditUserTag, requires(permission.EditUserTag))
						apiUsersUIDTagsTID.DELETE("", h.RemoveUserTag, requires(permission.EditUserTag))
					}
				}
			}
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", h.GetMe, requires(permission.GetMe))
				apiUsersMe.PATCH("", h.EditMe, requires(permission.EditMe))
				apiUsersMe.GET("/oidc", h.GetMeOIDC, requires(permission.GetOIDCUserInfo))
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.GetUserQRCode), blockBot)
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.ChangeMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.PUT("/password", h.PutMyPassword, requires(permission.ChangeMyPassword), blockBot)
				apiUsersMe.POST("/fcm-device", h.PostMyFCMDevice, requires(permission.RegisterFCMDevice), blockBot)
				apiUsersMe.GET("/view-states", h.GetMyViewStates, requires(permission.ConnectNotificationStream), blockBot)
				apiUsersMeTags := apiUsersMe.Group("/tags")
				{
					apiUsersMeTags.GET("", h.GetMyUserTags, requires(permission.GetUserTag))
					apiUsersMeTags.POST("", h.AddMyUserTag, requires(permission.EditUserTag))
					apiUsersMeTagsTID := apiUsersMeTags.Group("/:tagID")
					{
						apiUsersMeTagsTID.PATCH("", h.EditMyUserTag, requires(permission.EditUserTag))
						apiUsersMeTagsTID.DELETE("", h.RemoveMyUserTag, requires(permission.EditUserTag))
					}
				}
				apiUsersMeStars := apiUsersMe.Group("/stars", blockBot)
				{
					apiUsersMeStars.GET("", h.GetMyStars, requires(permission.GetChannelStar))
					apiUsersMeStars.POST("", h.PostStar, requires(permission.EditChannelStar))
					apiUsersMeStars.DELETE("/:channelID", h.RemoveMyStar, requires(permission.EditChannelStar))
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread", blockBot)
				{
					apiUsersMeUnread.GET("", h.GetMyUnreadChannels, requires(permission.GetUnread))
					apiUsersMeUnread.DELETE("/:channelID", h.ReadChannel, requires(permission.DeleteUnread))
				}
				apiUsersMeSubscriptions := apiUsersMe.Group("/subscriptions", blockBot)
				{
					apiUsersMeSubscriptions.GET("", h.GetMyChannelSubscriptions, requires(permission.GetChannelSubscription))
					apiUsersMeSubscriptions.PUT("/:channelID", h.SetChannelSubscribeLevel, requires(permission.EditChannelSubscription))
				}
				apiUsersMeSessions := apiUsersMe.Group("/sessions", blockBot)
				{
					apiUsersMeSessions.GET("", h.GetMySessions, requires(permission.GetMySessions))
					apiUsersMeSessions.DELETE("/:referenceID", h.RevokeMySession, requires(permission.DeleteMySessions))
				}
				apiUsersMeTokens := apiUsersMe.Group("/tokens", blockBot)
				{
					apiUsersMeTokens.GET("", h.GetMyTokens, requires(permission.GetMyTokens))
					apiUsersMeTokens.DELETE("/:tokenID", h.RevokeMyToken, requires(permission.RevokeMyToken))
				}
				apiUsersMeExAccounts := apiUsersMe.Group("/ex-accounts", blockBot)
				{
					apiUsersMeExAccounts.GET("", h.GetMyExternalAccounts, requires(permission.GetMyExternalAccount))
					apiUsersMeExAccounts.POST("/link", h.LinkExternalAccount, requires(permission.EditMyExternalAccount))
					apiUsersMeExAccounts.POST("/unlink", h.UnlinkExternalAccount, requires(permission.EditMyExternalAccount))
				}
				apiUsersMeSettings := apiUsersMe.Group("/settings", blockBot)
				{
					apiUsersMeSettings.GET("", h.GetMySettings, requires(permission.GetMe))
					apiUsersMeSettings.GET("/notify-citation", h.GetMyNotifyCitation, requires(permission.GetMe))
					apiUsersMeSettings.PUT("/notify-citation", h.PutMyNotifyCitation, requires(permission.EditMe))
				}
			}
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", h.GetChannels, requires(permission.GetChannel))
			apiChannels.POST("", h.CreateChannels, requires(permission.CreateChannel))
			apiChannelsCID := apiChannels.Group("/:channelID", retrieve.ChannelID(), requiresChannelAccessPerm)
			{
				apiChannelsCID.GET("", h.GetChannel, requires(permission.GetChannel))
				apiChannelsCID.PATCH("", h.EditChannel, requires(permission.EditChannel))
				apiChannelsCID.GET("/messages", h.GetMessages, requires(permission.GetMessage))
				apiChannelsCID.POST("/messages", h.PostMessage, bodyLimit(100), requires(permission.PostMessage))
				apiChannelsCID.GET("/stats", h.GetChannelStats, requires(permission.GetChannel))
				apiChannelsCID.GET("/topic", h.GetChannelTopic, requires(permission.GetChannel))
				apiChannelsCID.PUT("/topic", h.EditChannelTopic, requires(permission.EditChannelTopic))
				apiChannelsCID.GET("/viewers", h.GetChannelViewers, requires(permission.GetChannel))
				apiChannelsCID.GET("/pins", h.GetChannelPins, requires(permission.GetMessage))
				apiChannelsCID.GET("/subscribers", h.GetChannelSubscribers, requires(permission.GetChannelSubscription))
				apiChannelsCID.PUT("/subscribers", h.SetChannelSubscribers, requires(permission.EditChannelSubscription))
				apiChannelsCID.PATCH("/subscribers", h.EditChannelSubscribers, requires(permission.EditChannelSubscription))
				apiChannelsCID.GET("/bots", h.GetChannelBots, requires(permission.GetChannel))
				apiChannelsCID.GET("/events", h.GetChannelEvents, requires(permission.GetChannel))
			}
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("", h.SearchMessages, requires(permission.GetMessage))
			apiMessagesMID := apiMessages.Group("/:messageID", retrieve.MessageID(), requiresMessageAccessPerm)
			{
				apiMessagesMID.GET("", h.GetMessage, requires(permission.GetMessage))
				apiMessagesMID.PUT("", h.EditMessage, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMID.DELETE("", h.DeleteMessage, requires(permission.DeleteMessage))
				apiMessagesMID.GET("/pin", h.GetPin, requires(permission.GetMessage))
				apiMessagesMID.POST("/pin", h.CreatePin, requires(permission.CreateMessagePin))
				apiMessagesMID.DELETE("/pin", h.RemovePin, requires(permission.DeleteMessagePin))
				apiMessagesMID.GET("/clips", h.GetMessageClips, requires(permission.GetClipFolder))
				apiMessagesMIDStamps := apiMessagesMID.Group("/stamps")
				{
					apiMessagesMIDStamps.GET("", h.GetMessageStamps, requires(permission.GetMessage))
					apiMessagesMIDStampsSID := apiMessagesMIDStamps.Group("/:stampID", retrieve.StampID(true))
					{
						apiMessagesMIDStampsSID.POST("", h.AddMessageStamp, requires(permission.AddMessageStamp))
						apiMessagesMIDStampsSID.DELETE("", h.RemoveMessageStamp, requires(permission.RemoveMessageStamp))
					}
				}
			}
		}
		apiFiles := api.Group("/files")
		{
			apiFiles.GET("", h.GetFiles, requires(permission.DownloadFile))
			apiFiles.POST("", h.PostFile, bodyLimit(30<<10), requires(permission.UploadFile))
			apiFilesFID := apiFiles.Group("/:fileID", retrieve.FileID(), requiresFileAccessPerm)
			{
				apiFilesFID.GET("", h.GetFile, requires(permission.DownloadFile))
				apiFilesFID.DELETE("", h.DeleteFile, requires(permission.DeleteFile))
				apiFilesFID.GET("/meta", h.GetFileMeta, requires(permission.DownloadFile))
				apiFilesFID.GET("/thumbnail", h.GetThumbnailImage, requires(permission.DownloadFile))
			}
		}
		apiTags := api.Group("/tags")
		{
			apiTagsTID := apiTags.Group("/:tagID")
			{
				apiTagsTID.GET("", h.GetTag, requires(permission.GetUserTag))
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", h.GetStamps, requires(permission.GetStamp))
			apiStamps.POST("", h.CreateStamp, requires(permission.CreateStamp))
			apiStampsSID := apiStamps.Group("/:stampID", retrieve.StampID(false))
			{
				apiStampsSID.GET("", h.GetStamp, requires(permission.GetStamp))
				apiStampsSID.PATCH("", h.EditStamp, requires(permission.EditStamp))
				apiStampsSID.DELETE("", h.DeleteStamp, requiresDeleteStampPerm)
				apiStampsSID.GET("/stats", h.GetStampStats, requires(permission.GetStamp))
				apiStampsSID.GET("/image", h.GetStampImage, requires(permission.GetStamp, permission.DownloadFile))
				apiStampsSID.PUT("/image", h.ChangeStampImage, requires(permission.EditStamp))
			}
		}
		apiStampPalettes := api.Group("/stamp-palettes", blockBot)
		{
			apiStampPalettes.GET("", h.GetStampPalettes, requires(permission.GetStampPalette))
			apiStampPalettes.POST("", h.CreateStampPalette, requires(permission.CreateStampPalette))
			apiStampPalettesPID := apiStampPalettes.Group("/:paletteID", retrieve.StampPalettesID())
			{
				apiStampPalettesPID.GET("", h.GetStampPalette, requires(permission.GetStampPalette))
				apiStampPalettesPID.PATCH("", h.EditStampPalette, requires(permission.EditStampPalette))
				apiStampPalettesPID.DELETE("", h.DeleteStampPalette, requires(permission.DeleteStampPalette))
			}
		}
		apiWebhooks := api.Group("/webhooks", blockBot)
		{
			apiWebhooks.GET("", h.GetWebhooks, requires(permission.GetWebhook))
			apiWebhooks.POST("", h.CreateWebhook, requires(permission.CreateWebhook))
			apiWebhooksWID := apiWebhooks.Group("/:webhookID", retrieve.WebhookID(), requiresWebhookAccessPerm)
			{
				apiWebhooksWID.GET("", h.GetWebhook, requires(permission.GetWebhook))
				apiWebhooksWID.PATCH("", h.EditWebhook, requires(permission.EditWebhook))
				apiWebhooksWID.DELETE("", h.DeleteWebhook, requires(permission.DeleteWebhook))
				apiWebhooksWID.GET("/icon", h.GetWebhookIcon, requires(permission.GetWebhook))
				apiWebhooksWID.PUT("/icon", h.ChangeWebhookIcon, requires(permission.EditWebhook))
				apiWebhooksWID.GET("/messages", h.GetWebhookMessages, requires(permission.GetWebhook))
			}
		}
		apiGroups := api.Group("/groups")
		{
			apiGroups.GET("", h.GetUserGroups, requires(permission.GetUserGroup))
			apiGroups.POST("", h.PostUserGroups, requires(permission.CreateUserGroup))
			apiGroupsGID := apiGroups.Group("/:groupID", retrieve.GroupID())
			{
				apiGroupsGID.GET("", h.GetUserGroup, requires(permission.GetUserGroup))
				apiGroupsGID.PATCH("", h.EditUserGroup, requiresGroupAdminPerm, requires(permission.EditUserGroup))
				apiGroupsGID.DELETE("", h.DeleteUserGroup, requiresGroupAdminPerm, requires(permission.DeleteUserGroup))
				apiGroupsGIDMembers := apiGroupsGID.Group("/members")
				{
					apiGroupsGIDMembers.GET("", h.GetUserGroupMembers, requires(permission.GetUserGroup))
					apiGroupsGIDMembers.POST("", h.AddUserGroupMember, requiresGroupAdminPerm, requires(permission.EditUserGroup))
					apiGroupsGIDMembersUID := apiGroupsGIDMembers.Group("/:userID", requiresGroupAdminPerm)
					{
						apiGroupsGIDMembersUID.PATCH("", h.EditUserGroupMember, requires(permission.EditUserGroup))
						apiGroupsGIDMembersUID.DELETE("", h.RemoveUserGroupMember, requires(permission.EditUserGroup))
					}
				}
				apiGroupsGIDAdmins := apiGroupsGID.Group("/admins")
				{
					apiGroupsGIDAdmins.GET("", h.GetUserGroupAdmins, requires(permission.GetUserGroup))
					apiGroupsGIDAdmins.POST("", h.AddUserGroupAdmin, requiresGroupAdminPerm, requires(permission.EditUserGroup))
					apiGroupsGIDAdminsUID := apiGroupsGIDAdmins.Group("/:userID", requiresGroupAdminPerm)
					{
						apiGroupsGIDAdminsUID.DELETE("", h.RemoveUserGroupAdmin, requires(permission.EditUserGroup))
					}
				}
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/timeline", h.GetActivityTimeline, requires(permission.GetMessage))
			apiActivity.GET("/onlines", h.GetOnlineUsers, requires(permission.GetUser))
		}
		apiClients := api.Group("/clients", blockBot)
		{
			apiClients.GET("", h.GetClients, requires(permission.GetClients))
			apiClients.POST("", h.CreateClient, requires(permission.CreateClient))
			apiClientsCID := apiClients.Group("/:clientID", retrieve.ClientID())
			{
				apiClientsCID.GET("", h.GetClient, requires(permission.GetClients))
				apiClientsCID.PATCH("", h.EditClient, requiresClientAccessPerm, requires(permission.EditMyClient))
				apiClientsCID.DELETE("", h.DeleteClient, requiresClientAccessPerm, requires(permission.DeleteMyClient))
				apiClientsCID.DELETE("/tokens", h.RevokeClientTokens, requires(permission.RevokeMyToken))
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.CreateBot, requires(permission.CreateBot))
			apiBots.GET("/ws", echo.WrapHandler(h.BotWS), blockNonBot)
			apiBotsBID := apiBots.Group("/:botID", retrieve.BotID())
			{
				apiBotsBID.GET("", h.GetBot, requires(permission.GetBot))
				apiBotsBID.PATCH("", h.EditBot, requiresBotAccessPerm, requires(permission.EditBot))
				apiBotsBID.DELETE("", h.DeleteBot, requiresBotAccessPerm, requires(permission.DeleteBot))
				apiBotsBID.GET("/icon", h.GetBotIcon, requires(permission.GetBot))
				apiBotsBID.PUT("/icon", h.ChangeBotIcon, requiresBotAccessPerm, requires(permission.EditBot))
				apiBotsBID.GET("/logs", h.GetBotLogs, requiresBotAccessPerm, requires(permission.GetBot))
				apiBotsBIDActions := apiBotsBID.Group("/actions", requiresBotAccessPerm)
				{
					apiBotsBIDActions.POST("/activate", h.ActivateBot, requires(permission.EditBot))
					apiBotsBIDActions.POST("/inactivate", h.InactivateBot, requires(permission.EditBot))
					apiBotsBIDActions.POST("/reissue", h.ReissueBot, requires(permission.EditBot))
					apiBotsBIDActions.POST("/join", h.LetBotJoinChannel, requires(permission.BotActionJoinChannel))
					apiBotsBIDActions.POST("/leave", h.LetBotLeaveChannel, requires(permission.BotActionLeaveChannel))
				}
			}
		}
		apiWebRTC := api.Group("/webrtc", requires(permission.WebRTC))
		{
			apiWebRTC.GET("/state", h.GetWebRTCState)
			apiWebRTC.POST("/authenticate", h.PostWebRTCAuthenticate)
		}
		apiClipFolders := api.Group("/clip-folders", blockBot)
		{
			apiClipFolders.GET("", h.GetClipFolders, requires(permission.GetClipFolder))
			apiClipFolders.POST("", h.CreateClipFolder, requires(permission.CreateClipFolder))
			apiClipFoldersFID := apiClipFolders.Group("/:folderID", retrieve.ClipFolderID(), requiresClipFolderAccessPerm)
			{
				apiClipFoldersFID.GET("", h.GetClipFolder, requires(permission.GetClipFolder))
				apiClipFoldersFID.PATCH("", h.EditClipFolder, requires(permission.EditClipFolder))
				apiClipFoldersFID.DELETE("", h.DeleteClipFolder, requires(permission.DeleteClipFolder))
				apiClipFoldersFIDMessages := apiClipFoldersFID.Group("/messages")
				{
					apiClipFoldersFIDMessages.GET("", h.GetClipFolderMessages, requires(permission.GetClipFolder, permission.GetMessage))
					apiClipFoldersFIDMessages.POST("", h.PostClipFolderMessage, requires(permission.EditClipFolder))
					apiClipFoldersFIDMessages.DELETE("/:messageID", h.DeleteClipFolderMessages, requires(permission.EditClipFolder))
				}
			}
		}
		apiOgp := api.Group("/ogp", blockBot)
		{
			apiOgp.GET("", h.GetOgp)
			apiOgp.DELETE("/cache", h.DeleteOgpCache)
		}
		api.GET("/ws", echo.WrapHandler(h.WS), requires(permission.ConnectNotificationStream), blockBot)
	}

	apiNoAuth := e.Group("/v3")
	{
		apiNoAuth.GET("/version", h.GetVersion)
		apiNoAuth.GET("/jwks", h.GetJWKS)
		if h.Config.AllowSignUp {
			apiNoAuth.POST("/users", h.CreateUser, noLogin)
		}
		apiNoAuth.POST("/login", h.Login, noLogin)
		apiNoAuth.POST("/logout", h.Logout)
		apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook, retrieve.WebhookID())
		apiNoAuthPublic := apiNoAuth.Group("/public")
		{
			apiNoAuthPublic.GET("/icon/:username", h.GetPublicUserIcon)
		}
	}
}

// L ロガーを返します
func (h *Handlers) L(c echo.Context) *zap.Logger {
	return h.Logger.With(zap.String("requestId", extension.GetRequestID(c)))
}
