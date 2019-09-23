package router

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils/validator"
)

// SetupRouting APIルーティングを行います
func SetupRouting(e *echo.Echo, h *Handlers) {
	e.Validator = validator.New()
	e.Use(RequestCounterMiddleware())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{"X-TRAQ-VERSION", headerCacheFile, headerFileMetaType, headerMore},
		AllowHeaders:  []string{echo.HeaderContentType, echo.HeaderAuthorization, headerSignature},
		MaxAge:        3600,
	}))

	// middleware preparation
	requires := AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := RequestBodyLengthLimit
	botGuard := h.BotGuard
	only := func(role string) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// ユーザーロール検証
				user := c.Get("user").(*model.User)
				if user.Role != role {
					return forbidden(fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
				}
				return next(c) // OK
			}
		}
	}

	api := e.Group("/api/1.0", h.UserAuthenticate())
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", h.GetUsers, requires(permission.GetUser))
			apiUsers.POST("", h.PostUsers, requires(permission.RegisterUser))
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", h.GetMe, requires(permission.GetMe))
				apiUsersMe.PATCH("", h.PatchMe, requires(permission.EditMe))
				apiUsersMe.PUT("/password", h.PutPassword, requires(permission.ChangeMyPassword))
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.GetUserQRCode))
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.PutMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/groups", h.GetMyBelongingGroup, requires(permission.GetUserGroup))
				apiUsersMe.GET("/notification", h.GetMyNotificationChannels, requires(permission.GetChannelSubscription))
				apiUsersMe.GET("/tokens", h.GetMyTokens, requires(permission.GetMyTokens))
				apiUsersMe.DELETE("/tokens/:tokenID", h.DeleteMyToken, requires(permission.RevokeMyToken))
				apiUsersMeSessions := apiUsersMe.Group("/sessions")
				{
					apiUsersMeSessions.GET("", h.GetMySessions, requires(permission.GetMySessions))
					apiUsersMeSessions.DELETE("", h.DeleteAllMySessions, requires(permission.DeleteMySessions))
					apiUsersMeSessions.DELETE("/:referenceID", h.DeleteMySession, requires(permission.DeleteMySessions))
				}
				apiUsersMeClips := apiUsersMe.Group("/clips")
				{
					apiUsersMeClips.GET("", h.GetClips, requires(permission.GetClip))
					apiUsersMeClips.POST("", h.PostClip, requires(permission.CreateClip))
					apiUsersMeClipsCid := apiUsersMeClips.Group("/:clipID", h.ValidateClipID())
					{
						apiUsersMeClipsCid.GET("", h.GetClip, requires(permission.GetClip))
						apiUsersMeClipsCid.DELETE("", h.DeleteClip, requires(permission.DeleteClip))
						apiUsersMeClipsCid.GET("/folder", h.GetClipsFolder, requires(permission.GetClip, permission.GetClipFolder))
						apiUsersMeClipsCid.PUT("/folder", h.PutClipsFolder, requires(permission.CreateClip))
					}
					apiUsersMeClipsFolders := apiUsersMeClips.Group("/folders")
					{
						apiUsersMeClipsFolders.GET("", h.GetClipFolders, requires(permission.GetClipFolder))
						apiUsersMeClipsFolders.POST("", h.PostClipFolder, requires(permission.CreateClipFolder))
						apiUsersMeClipsFoldersFid := apiUsersMeClipsFolders.Group("/:folderID", h.ValidateClipFolderID())
						{
							apiUsersMeClipsFoldersFid.GET("", h.GetClipFolder, requires(permission.GetClip, permission.GetClipFolder))
							apiUsersMeClipsFoldersFid.PATCH("", h.PatchClipFolder, requires(permission.PatchClipFolder))
							apiUsersMeClipsFoldersFid.DELETE("", h.DeleteClipFolder, requires(permission.DeleteClipFolder))
						}
					}
				}
				apiUsersMeStars := apiUsersMe.Group("/stars")
				{
					apiUsersMeStars.GET("", h.GetStars, requires(permission.GetChannelStar))
					apiUsersMeStarsCid := apiUsersMeStars.Group("/:channelID", h.ValidateChannelID(true))
					{
						apiUsersMeStarsCid.PUT("", h.PutStars, requires(permission.EditChannelStar))
						apiUsersMeStarsCid.DELETE("", h.DeleteStars, requires(permission.EditChannelStar))
					}
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread")
				{
					apiUsersMeUnread.GET("/channels", h.GetUnreadChannels, requires(permission.GetUnread))
					apiUsersMeUnread.DELETE("/channels/:channelID", h.DeleteUnread, requires(permission.DeleteUnread))
				}
				apiUsersMeMute := apiUsersMe.Group("/mute")
				{
					apiUsersMeMute.GET("", h.GetMutedChannelIDs, requires(permission.GetChannelMute))
					apiUsersMeMuteCid := apiUsersMeMute.Group("/:channelID", h.ValidateChannelID(false))
					{
						apiUsersMeMuteCid.POST("", h.PostMutedChannel, requires(permission.EditChannelMute))
						apiUsersMeMuteCid.DELETE("", h.DeleteMutedChannel, requires(permission.EditChannelMute))
					}
				}
				apiUsersMeFavoriteStamps := apiUsersMe.Group("/favorite-stamps")
				{
					apiUsersMeFavoriteStamps.GET("", h.GetMyFavoriteStamps, requires(permission.GetFavoriteStamp))
					apiUsersMeFavoriteStamps.POST("", h.PostMyFavoriteStamp, requires(permission.EditFavoriteStamp))
					apiUsersMeFavoriteStamps.DELETE("/:stampID", h.DeleteMyFavoriteStamp, requires(permission.EditFavoriteStamp))
				}
			}
			apiUsersUID := apiUsers.Group("/:userID", h.ValidateUserID(false))
			{
				apiUsersUID.GET("", h.GetUserByID, requires(permission.GetUser))
				apiUsersUID.PATCH("", h.PatchUserByID, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/status", h.PutUserStatus, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/password", h.PutUserPassword, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/messages", h.GetDirectMessages, requires(permission.GetMessage), botGuard(blockUnlessSubscribingEvent(bot.DirectMessageCreated)))
				apiUsersUID.POST("/messages", h.PostDirectMessage, bodyLimit(100), requires(permission.PostMessage), botGuard(blockUnlessSubscribingEvent(bot.DirectMessageCreated)))
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
			apiHeartBeat.GET("", h.GetHeartbeat, requires(permission.GetHeartbeat))
			apiHeartBeat.POST("", h.PostHeartbeat, requires(permission.PostHeartbeat))
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", h.GetChannels, requires(permission.GetChannel))
			apiChannels.POST("", h.PostChannels, requires(permission.CreateChannel))
			apiChannelsCid := apiChannels.Group("/:channelID", h.ValidateChannelID(false))
			{
				apiChannelsCid.GET("", h.GetChannelByChannelID, requires(permission.GetChannel))
				apiChannelsCid.PATCH("", h.PatchChannelByChannelID, requires(permission.EditChannel))
				apiChannelsCid.DELETE("", h.DeleteChannelByChannelID, requires(permission.DeleteChannel))
				apiChannelsCid.PUT("/parent", h.PutChannelParent, requires(permission.ChangeParentChannel))
				apiChannelsCid.POST("/children", h.PostChannelChildren, requires(permission.CreateChannel))
				apiChannelsCid.GET("/pins", h.GetChannelPin, requires(permission.GetMessage))
				apiChannelsCid.GET("/events", h.GetChannelEvents, requires(permission.GetChannel))
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
				apiChannelsCidBots := apiChannelsCid.Group("/bots")
				{
					apiChannelsCidBots.GET("", h.GetChannelBots, requires(permission.GetBot))
					apiChannelsCidBots.POST("", h.PostChannelBots, requires(permission.InstallBot))
					apiChannelsCidBots.DELETE("/:botID", h.DeleteChannelBot, requires(permission.UninstallBot), h.ValidateBotID(false))
				}
				apiChannelsCidWebRTC := apiChannelsCid.Group("/webrtc")
				{
					apiChannelsCidWebRTC.GET("/state", h.GetChannelWebRTCState, requires(permission.GetChannel))
				}
			}
		}
		apiNotification := api.Group("/notification")
		{
			apiNotification.GET("", echo.WrapHandler(h.SSE), requires(permission.ConnectNotificationStream))
			apiNotification.POST("/device", h.PostDeviceToken, requires(permission.RegisterFCMDevice))
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("/reports", h.GetMessageReports, requires(permission.GetMessageReports))
			apiMessagesMid := apiMessages.Group("/:messageID", h.ValidateMessageID())
			{
				apiMessagesMid.GET("", h.GetMessageByID, requires(permission.GetMessage))
				apiMessagesMid.PUT("", h.PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMid.DELETE("", h.DeleteMessageByID, requires(permission.DeleteMessage))
				apiMessagesMid.POST("/report", h.PostMessageReport, requires(permission.ReportMessage))
				apiMessagesMid.GET("/stamps", h.GetMessageStamps, requires(permission.GetMessage))
				apiMessagesMidStampsSid := apiMessagesMid.Group("/stamps/:stampID", h.ValidateStampID(true))
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
			apiFiles.POST("", h.PostFile, bodyLimit(30<<10), requires(permission.UploadFile))
			apiFilesFid := apiFiles.Group("/:fileID", h.ValidateFileID())
			{
				apiFilesFid.GET("", h.GetFileByID, requires(permission.DownloadFile))
				apiFilesFid.DELETE("", h.DeleteFileByID, requires(permission.DeleteFile))
				apiFilesFid.GET("/meta", h.GetMetaDataByFileID, requires(permission.DownloadFile))
				apiFilesFid.GET("/thumbnail", h.GetThumbnailByID, requires(permission.DownloadFile))
			}
		}
		apiPins := api.Group("/pins")
		{
			apiPins.POST("", h.PostPin, requires(permission.CreateMessagePin))
			apiPinsPid := apiPins.Group("/:pinID", h.ValidatePinID())
			{
				apiPinsPid.GET("", h.GetPin, requires(permission.GetMessage))
				apiPinsPid.DELETE("", h.DeletePin, requires(permission.DeleteMessagePin))
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", h.GetStamps, requires(permission.GetStamp))
			apiStamps.POST("", h.PostStamp, requires(permission.CreateStamp))
			apiStampsSid := apiStamps.Group("/:stampID", h.ValidateStampID(false))
			{
				apiStampsSid.GET("", h.GetStamp, requires(permission.GetStamp))
				apiStampsSid.PATCH("", h.PatchStamp, requires(permission.EditStamp))
				apiStampsSid.DELETE("", h.DeleteStamp, requires(permission.DeleteStamp))
			}
		}
		apiWebhooks := api.Group("/webhooks")
		{
			apiWebhooks.GET("", h.GetWebhooks, requires(permission.GetWebhook))
			apiWebhooks.POST("", h.PostWebhooks, requires(permission.CreateWebhook))
			apiWebhooksWid := apiWebhooks.Group("/:webhookID", h.ValidateWebhookID(true))
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
			apiGroupsGid := apiGroups.Group("/:groupID", h.ValidateGroupID())
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
		apiClients := api.Group("/clients")
		{
			apiClients.GET("", h.GetClients, requires(permission.GetClients))
			apiClients.POST("", h.PostClients, requires(permission.CreateClient))
			apiClientCid := apiClients.Group("/:clientID")
			{
				apiClientCid.GET("", h.GetClient, requires(permission.GetClients), h.ValidateClientID(false))
				apiClientCid.PATCH("", h.PatchClient, requires(permission.EditMyClient), h.ValidateClientID(true))
				apiClientCid.DELETE("", h.DeleteClient, requires(permission.DeleteMyClient), h.ValidateClientID(true))
				apiClientCid.GET("/detail", h.GetClientDetail, requires(permission.GetClients), h.ValidateClientID(true))
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.PostBots, requires(permission.CreateBot))
			apiBotsBid := apiBots.Group("/:botID")
			{
				apiBotsBid.GET("", h.GetBot, requires(permission.GetBot), h.ValidateBotID(false))
				apiBotsBid.PATCH("", h.PatchBot, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.DELETE("", h.DeleteBot, requires(permission.DeleteBot), h.ValidateBotID(true))
				apiBotsBid.GET("/detail", h.GetBotDetail, requires(permission.GetBot), h.ValidateBotID(true))
				apiBotsBid.PUT("/events", h.PutBotEvents, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.GET(`/events/logs`, h.GetBotEventLogs, requires(permission.GetBot), h.ValidateBotID(true))
				apiBotsBid.GET("/icon", h.GetBotIcon, requires(permission.GetBot), h.ValidateBotID(false))
				apiBotsBid.PUT("/icon", h.PutBotIcon, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.PUT("/state", h.PutBotState, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.POST("/reissue", h.PostBotReissueTokens, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.GET("/channels", h.GetBotJoinChannels, requires(permission.GetBot), h.ValidateBotID(true))
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/latest-messages", h.GetActivityLatestMessages, requires(permission.GetMessage))
		}
		apiAuthority := api.Group("/authority", only(role.Admin))
		{
			apiAuthorityRoles := apiAuthority.Group("/roles")
			{
				apiAuthorityRoles.GET("", h.GetRoles)
				apiAuthorityRoles.POST("", h.PostRoles)
				apiAuthorityRolesRid := apiAuthorityRoles.Group("/:role")
				{
					apiAuthorityRolesRid.GET("", h.GetRole)
					apiAuthorityRolesRid.PATCH("", h.PatchRole)
				}
			}
			apiAuthority.GET("/permissions", h.GetPermissions)
			apiAuthority.GET("/reload", h.GetAuthorityReload)
			apiAuthority.POST("/reload", h.PostAuthorityReload)
		}
		apiWebRTC := api.Group("/webrtc")
		{
			apiWebRTC.GET("/state", h.GetWebRTCState)
			apiWebRTC.PUT("/state", h.PutWebRTCState)
		}
		api.POST("/oauth2/authorize/decide", h.AuthorizationDecideHandler, botGuard(blockAlways))

		if len(h.SkyWaySecretKey) > 0 {
			api.POST("/skyway/authenticate", h.PostSkyWayAuthenticate, botGuard(blockAlways))
		}
	}

	apiNoAuth := e.Group("/api/1.0")
	{
		apiNoAuth.POST("/login", h.PostLogin)
		apiNoAuth.POST("/logout", h.PostLogout)
		apiPublic := apiNoAuth.Group("/public")
		{
			apiPublic.GET("/icon/:username", h.GetPublicUserIcon)
			apiPublic.GET("/emoji.json", h.GetPublicEmojiJSON)
			apiPublic.GET("/emoji.css", h.GetPublicEmojiCSS)
			apiPublic.GET("/emoji/:stampID", h.GetPublicEmojiImage, h.ValidateStampID(false))
		}
		apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook, h.ValidateWebhookID(false))
		apiNoAuth.POST("/webhooks/:webhookID/github", h.PostWebhookByGithub, h.ValidateWebhookID(false))
		apiOAuth := apiNoAuth.Group("/oauth2")
		{
			apiOAuth.GET("/authorize", h.AuthorizationEndpointHandler)
			apiOAuth.POST("/authorize", h.AuthorizationEndpointHandler)
			apiOAuth.POST("/token", h.TokenEndpointHandler)
			apiOAuth.POST("/revoke", h.RevokeTokenEndpointHandler)
		}
		apiNoAuth.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	}
}
