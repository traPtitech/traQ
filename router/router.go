package router

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/utils/validator"
)

// SetupRouting APIルーティングを行います
func SetupRouting(e *echo.Echo, h *Handlers) {
	e.Validator = validator.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{"X-TRAQ-VERSION", headerCacheFile, headerFileMetaType},
		AllowHeaders:  []string{echo.HeaderContentType, echo.HeaderAuthorization, headerSignature},
		MaxAge:        3600,
	}))

	// middleware preparation
	requires := AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := RequestBodyLengthLimit
	botGuard := h.BotGuard

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
				apiUsersMe.PUT("/password", h.PutPassword, requires(permission.ChangeMyPassword), botGuard(blockAlways))
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.DownloadFile))
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.PutMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/groups", h.GetMyBelongingGroup)
				apiUsersMe.GET("/notification", h.GetMyNotificationChannels, requires(permission.GetNotificationStatus), botGuard(blockAlways))
				apiUsersMe.GET("/tokens", h.GetMyTokens, requires(permission.GetMyTokens), botGuard(blockAlways))
				apiUsersMe.DELETE("/tokens/:tokenID", h.DeleteMyToken, requires(permission.RevokeMyToken), botGuard(blockAlways))
				apiUsersMeSessions := apiUsersMe.Group("/sessions", botGuard(blockAlways))
				{
					apiUsersMeSessions.GET("", h.GetMySessions, requires(permission.GetMySessions))
					apiUsersMeSessions.DELETE("", h.DeleteAllMySessions, requires(permission.DeleteMySessions))
					apiUsersMeSessions.DELETE("/:referenceID", h.DeleteMySession, requires(permission.DeleteMySessions))
				}
				apiUsersMeClips := apiUsersMe.Group("/clips", botGuard(blockAlways))
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
				apiUsersMeStars := apiUsersMe.Group("/stars", botGuard(blockAlways))
				{
					apiUsersMeStars.GET("", h.GetStars, requires(permission.GetStar))
					apiUsersMeStarsCid := apiUsersMeStars.Group("/:channelID", h.ValidateChannelID(true))
					{
						apiUsersMeStarsCid.PUT("", h.PutStars, requires(permission.CreateStar))
						apiUsersMeStarsCid.DELETE("", h.DeleteStars, requires(permission.DeleteStar))
					}
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread", botGuard(blockAlways))
				{
					apiUsersMeUnread.GET("", h.GetUnread, requires(permission.GetUnread))
					apiUsersMeUnread.DELETE("/:channelID", h.DeleteUnread, requires(permission.DeleteUnread))
				}
				apiUsersMeMute := apiUsersMe.Group("/mute", botGuard(blockAlways))
				{
					apiUsersMeMute.GET("", h.GetMutedChannelIDs, requires(permission.GetMutedChannels))
					apiUsersMeMuteCid := apiUsersMeMute.Group("/:channelID", h.ValidateChannelID(false))
					{
						apiUsersMeMuteCid.POST("", h.PostMutedChannel, requires(permission.MuteChannel))
						apiUsersMeMuteCid.DELETE("", h.DeleteMutedChannel, requires(permission.UnmuteChannel))
					}
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
				apiUsersUID.GET("/notification", h.GetNotificationChannels, requires(permission.GetNotificationStatus))
				apiUsersUID.GET("/groups", h.GetUserBelongingGroup)
				apiUsersUIDTags := apiUsersUID.Group("/tags")
				{
					apiUsersUIDTags.GET("", h.GetUserTags, requires(permission.GetTag))
					apiUsersUIDTags.POST("", h.PostUserTag, requires(permission.AddTag))
					apiUsersUIDTagsTid := apiUsersUIDTags.Group("/:tagID")
					{
						apiUsersUIDTagsTid.PATCH("", h.PatchUserTag, requires(permission.ChangeTagLockState))
						apiUsersUIDTagsTid.DELETE("", h.DeleteUserTag, requires(permission.RemoveTag))
					}
				}
			}
		}
		apiHeartBeat := api.Group("/heartbeat", botGuard(blockAlways))
		{
			apiHeartBeat.GET("", h.GetHeartbeat, requires(permission.GetHeartbeat))
			apiHeartBeat.POST("", h.PostHeartbeat, requires(permission.PostHeartbeat))
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", h.GetChannels, requires(permission.GetChannel), botGuard(blockAlways))
			apiChannels.POST("", h.PostChannels, requires(permission.CreateChannel), botGuard(blockAlways))
			apiChannelsCid := apiChannels.Group("/:channelID", h.ValidateChannelID(false), botGuard(blockByChannelIDQuery))
			{
				apiChannelsCid.GET("", h.GetChannelByChannelID, requires(permission.GetChannel))
				apiChannelsCid.PATCH("", h.PatchChannelByChannelID, requires(permission.EditChannel), botGuard(blockAlways))
				apiChannelsCid.DELETE("", h.DeleteChannelByChannelID, requires(permission.DeleteChannel), botGuard(blockAlways))
				apiChannelsCid.PUT("/parent", h.PutChannelParent, requires(permission.ChangeParentChannel), botGuard(blockAlways))
				apiChannelsCid.POST("/children", h.PostChannelChildren, requires(permission.CreateChannel), botGuard(blockAlways))
				apiChannelsCid.GET("/pins", h.GetChannelPin, requires(permission.GetPin))
				apiChannelsCidTopic := apiChannelsCid.Group("/topic")
				{
					apiChannelsCidTopic.GET("", h.GetTopic, requires(permission.GetTopic))
					apiChannelsCidTopic.PUT("", h.PutTopic, requires(permission.EditTopic))
				}
				apiChannelsCidMessages := apiChannelsCid.Group("/messages")
				{
					apiChannelsCidMessages.GET("", h.GetMessagesByChannelID, requires(permission.GetMessage))
					apiChannelsCidMessages.POST("", h.PostMessage, bodyLimit(100), requires(permission.PostMessage))
				}
				apiChannelsCidNotification := apiChannelsCid.Group("/notification")
				{
					apiChannelsCidNotification.GET("", h.GetNotificationStatus, requires(permission.GetNotificationStatus))
					apiChannelsCidNotification.PUT("", h.PutNotificationStatus, requires(permission.ChangeNotificationStatus), botGuard(blockAlways))
				}
				apiChannelsCidBots := apiChannelsCid.Group("/bots")
				{
					apiChannelsCidBots.GET("", h.GetChannelBots, requires(permission.GetBot))
					apiChannelsCidBots.POST("", h.PostChannelBots, requires(permission.InstallBot), botGuard(blockAlways))
					apiChannelsCidBots.DELETE("/:botID", h.DeleteChannelBot, requires(permission.UninstallBot), h.ValidateBotID(false), botGuard(blockAlways))
				}
			}
		}
		apiNotification := api.Group("/notification", botGuard(blockAlways))
		{
			apiNotification.GET("", h.NotificationStream, requires(permission.ConnectNotificationStream))
			apiNotification.POST("/device", h.PostDeviceToken, requires(permission.RegisterDevice))
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("/reports", h.GetMessageReports, requires(permission.GetMessageReports))
			apiMessagesMid := apiMessages.Group("/:messageID", h.ValidateMessageID(), botGuard(blockByMessageChannel))
			{
				apiMessagesMid.GET("", h.GetMessageByID, requires(permission.GetMessage))
				apiMessagesMid.PUT("", h.PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMid.DELETE("", h.DeleteMessageByID, requires(permission.DeleteMessage))
				apiMessagesMid.POST("/report", h.PostMessageReport, requires(permission.ReportMessage), botGuard(blockAlways))
				apiMessagesMid.GET("/stamps", h.GetMessageStamps, requires(permission.GetMessageStamp))
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
				apiTagsTid.GET("", h.GetUsersByTagID, requires(permission.GetTag))
				apiTagsTid.PATCH("", h.PatchTag, requires(permission.EditTag))
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
			apiPins.POST("", h.PostPin, requires(permission.CreatePin))
			apiPinsPid := apiPins.Group("/:pinID", h.ValidatePinID())
			{
				apiPinsPid.GET("", h.GetPin, requires(permission.GetPin))
				apiPinsPid.DELETE("", h.DeletePin, requires(permission.DeletePin))
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", h.GetStamps, requires(permission.GetStamp))
			apiStamps.POST("", h.PostStamp, requires(permission.CreateStamp), botGuard(blockAlways))
			apiStampsSid := apiStamps.Group("/:stampID", h.ValidateStampID(false))
			{
				apiStampsSid.GET("", h.GetStamp, requires(permission.GetStamp))
				apiStampsSid.PATCH("", h.PatchStamp, requires(permission.EditStamp), botGuard(blockAlways))
				apiStampsSid.DELETE("", h.DeleteStamp, requires(permission.DeleteStamp), botGuard(blockAlways))
			}
		}
		apiWebhooks := api.Group("/webhooks", botGuard(blockAlways))
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
			apiGroups.GET("", h.GetUserGroups)
			apiGroups.POST("", h.PostUserGroups)
			apiGroupsGid := apiGroups.Group("/:groupID", h.ValidateGroupID())
			{
				apiGroupsGid.GET("", h.GetUserGroup)
				apiGroupsGid.PATCH("", h.PatchUserGroup)
				apiGroupsGid.DELETE("", h.DeleteUserGroup)
				apiGroupsGidMembers := apiGroupsGid.Group("/members")
				{
					apiGroupsGidMembers.GET("", h.GetUserGroupMembers)
					apiGroupsGidMembers.POST("", h.PostUserGroupMembers)
					apiGroupsGidMembers.DELETE("/:userID", h.DeleteUserGroupMembers)
				}
			}
		}
		apiClients := api.Group("/clients", botGuard(blockAlways))
		{
			apiClients.GET("", h.GetClients, requires(permission.GetClients))
			apiClients.POST("", h.PostClients, requires(permission.CreateClient))
			apiClientCid := apiClients.Group("/:clientID")
			{
				apiClientCid.GET("", h.GetClient, requires(permission.GetClients), h.ValidateClientID(false))
				apiClientCid.PATCH("", h.PatchClient, requires(permission.EditMyClient), h.ValidateClientID(true))
				apiClientCid.DELETE("", h.DeleteClient, requires(permission.DeleteMyClient), h.ValidateClientID(true))
			}
		}
		apiBots := api.Group("/bots", botGuard(blockAlways))
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.PostBots, requires(permission.CreateBot))
			apiBotsBid := apiBots.Group("/:botID", h.ValidateBotID(false))
			{
				apiBotsBid.GET("", h.GetBot, requires(permission.GetBot))
				apiBotsBid.PATCH("", h.PatchBot, requires(permission.EditBot))
				apiBotsBid.DELETE("", h.DeleteBot, requires(permission.DeleteBot))
				apiBotsBid.GET("/detail", h.GetBotDetail, requires(permission.GetBot))
				apiBotsBid.PUT("/events", h.PutBotEvents, requires(permission.EditBot))
				apiBotsBid.GET(`/events/logs`, h.GetBotEventLogs, requires(permission.GetBot))
				apiBotsBid.GET("/icon", h.GetBotIcon, requires(permission.GetBot))
				apiBotsBid.PUT("/icon", h.PutBotIcon, requires(permission.EditBot))
				apiBotsBid.PUT("/state", h.PutBotState, requires(permission.EditBot))
				apiBotsBid.POST("/reissue", h.PostBotReissueTokens, requires(permission.EditBot))
				apiBotsBid.GET("/channels", h.GetBotJoinChannels, requires(permission.GetBot))
			}
		}
		apiActivity := api.Group("/activity", botGuard(blockAlways))
		{
			apiActivity.GET("/latest-messages", h.GetActivityLatestMessages, requires(permission.GetMessage))
		}
		api.POST("/oauth2/authorize/decide", h.AuthorizationDecideHandler, botGuard(blockAlways))
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
