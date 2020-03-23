package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/middlewares"
	"go.uber.org/zap"
)

type Handlers struct {
	RBAC     rbac.RBAC
	Repo     repository.Repository
	WS       *ws.Streamer
	Hub      *hub.Hub
	Logger   *zap.Logger
	Realtime *realtime.Service

	Version  string
	Revision string

	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := middlewares.RequestBodyLengthLimit
	retrieve := middlewares.NewParamRetriever(h.Repo)
	blockBot := middlewares.BlockBot(h.Repo)

	requiresBotAccessPerm := middlewares.CheckBotAccessPerm(h.RBAC, h.Repo)
	requiresWebhookAccessPerm := middlewares.CheckWebhookAccessPerm(h.RBAC, h.Repo)
	requiresFileAccessPerm := middlewares.CheckFileAccessPerm(h.RBAC, h.Repo)
	requiresClientAccessPerm := middlewares.CheckClientAccessPerm(h.RBAC, h.Repo)
	requiresMessageAccessPerm := middlewares.CheckMessageAccessPerm(h.RBAC, h.Repo)
	requiresChannelAccessPerm := middlewares.CheckChannelAccessPerm(h.RBAC, h.Repo)
	requiresGroupAdminPerm := middlewares.CheckUserGroupAdminPerm(h.RBAC, h.Repo)
	requiresClipFolderAccessPerm := middlewares.CheckClipFolderAccessPerm(h.RBAC, h.Repo)

	api := e.Group("/v3", middlewares.UserAuthenticate(h.Repo))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", h.GetUsers, requires(permission.GetUser))
			apiUsers.POST("", NotImplemented, requires(permission.RegisterUser))
			apiUsersUID := apiUsers.Group("/:userID", retrieve.UserID(false))
			{
				apiUsersUID.GET("", h.GetUser, requires(permission.GetUser))
				apiUsersUID.PATCH("", h.EditUser, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/messages", h.GetDirectMessages, requires(permission.GetMessage))
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
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.GetUserQRCode), blockBot)
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.ChangeMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.PUT("/password", h.PutMyPassword, requires(permission.ChangeMyPassword), blockBot)
				apiUsersMe.POST("/fcm-device", h.PostMyFCMDevice, requires(permission.RegisterFCMDevice), blockBot)
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
			apiMessagesMID := apiMessages.Group("/:messageID", retrieve.MessageID(), requiresMessageAccessPerm)
			{
				apiMessagesMID.GET("", h.GetMessage, requires(permission.GetMessage))
				apiMessagesMID.PUT("", h.EditMessage, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMID.DELETE("", h.DeleteMessage, requires(permission.DeleteMessage))
				apiMessagesMID.GET("/pin", h.GetPin, requires(permission.GetMessage))
				apiMessagesMID.POST("/pin", h.CreatePin, requires(permission.CreateMessagePin))
				apiMessagesMID.DELETE("/pin", h.RemovePin, requires(permission.DeleteMessagePin))
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
			apiFiles.GET("", NotImplemented, requires(permission.DownloadFile))
			apiFiles.POST("", h.PostFile, bodyLimit(30<<10), requires(permission.UploadFile))
			apiFilesFID := apiFiles.Group("/:fileID", retrieve.FileID(), requiresFileAccessPerm)
			{
				apiFilesFID.GET("", h.GetFile, requires(permission.DownloadFile))
				apiFilesFID.DELETE("", NotImplemented, requires(permission.DeleteFile))
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
				apiStampsSID.DELETE("", h.DeleteStamp, requires(permission.DeleteStamp))
				apiStampsSID.GET("/image", h.GetStampImage, requires(permission.GetStamp, permission.DownloadFile))
				apiStampsSID.PUT("/image", h.ChangeStampImage, requires(permission.EditStamp))
			}
		}
		apiStampPalettes := api.Group("/stamp-palettes", blockBot)
		{
			apiStampPalettes.GET("", NotImplemented, requires(permission.GetStampPalette))
			apiStampPalettes.POST("", NotImplemented, requires(permission.CreateStampPalette))
			apiStampPalettesPID := apiStampPalettes.Group("/:paletteID")
			{
				apiStampPalettesPID.GET("", NotImplemented, requires(permission.GetStampPalette))
				apiStampPalettesPID.PATCH("", NotImplemented, requires(permission.EditStampPalette))
				apiStampPalettesPID.DELETE("", NotImplemented, requires(permission.DeleteStampPalette))
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
				apiWebhooksWID.GET("/icon", h.GetWebhookIcon, requires(permission.GetWebhook, permission.DownloadFile))
				apiWebhooksWID.PUT("/icon", h.ChangeWebhookIcon, requires(permission.EditWebhook))
				apiWebhooksWID.GET("/messages", h.GetWebhookMessages, requires(permission.GetWebhook, permission.GetMessage))
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
			apiActivity.GET("/timelines", NotImplemented, requires(permission.GetMessage))
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
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.CreateBot, requires(permission.CreateBot))
			apiBotsBID := apiBots.Group("/:botID", retrieve.BotID())
			{
				apiBotsBID.GET("", h.GetBot, requires(permission.GetBot))
				apiBotsBID.PATCH("", h.EditBot, requiresBotAccessPerm, requires(permission.EditBot))
				apiBotsBID.DELETE("", h.DeleteBot, requiresBotAccessPerm, requires(permission.DeleteBot))
				apiBotsBID.GET("/icon", h.GetBotIcon, requires(permission.GetBot, permission.DownloadFile))
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
		apiWebRTC := api.Group("/webrtc", requires(permission.WebRTC), blockBot)
		{
			apiWebRTC.GET("/state", NotImplemented)
			apiWebRTC.POST("/authenticate", h.PostWebRTCAuthenticate)
		}
		apiClipFolders := api.Group("/clip-folders", blockBot)
		{
			apiClipFolders.GET("", h.GetClipFolders, requires(permission.GetClipFolder))
			apiClipFolders.POST("", h.CreateClipFolders, requires(permission.CreateClipFolder))
			apiClipFoldersFID := apiClipFolders.Group("/:folderID", retrieve.ClipFolderID(), requiresClipFolderAccessPerm)
			{
				apiClipFoldersFID.GET("", h.GetClipFolder, requires(permission.GetClipFolder))
				apiClipFoldersFID.PATCH("", h.EditClipFolder, requires(permission.EditClipFolder))
				apiClipFoldersFID.DELETE("", h.DeleteClipFolder, requires(permission.DeleteClipFolder))
				apiClipFoldersFIDMessages := apiClipFoldersFID.Group("/messages")
				{
					apiClipFoldersFIDMessages.GET("", h.GetClipFolderMessages, requires(permission.GetClipFolder, permission.GetMessage))
					apiClipFoldersFIDMessages.POST("", h.PostClipFolderMessages, requires(permission.EditClipFolder))
					apiClipFoldersFIDMessages.DELETE("/:messageID", h.DeleteClipFolderMessages, requires(permission.EditClipFolder), retrieve.MessageID())
				}
			}
		}
		api.GET("/ws", echo.WrapHandler(h.WS), requires(permission.ConnectNotificationStream), blockBot)
	}

	apiNoAuth := e.Group("/v3")
	{
		apiNoAuth.GET("/version", h.GetVersion)
		apiNoAuth.POST("/login", h.Login)
		apiNoAuth.POST("/logout", h.Logout)
		apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook)
		apiNoAuthPublic := apiNoAuth.Group("/public")
		{
			apiNoAuthPublic.GET("/icon/:username", h.GetPublicUserIcon)
		}
	}
}
