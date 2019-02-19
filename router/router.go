package router

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
)

// SetupRouting APIルーティングを行います
func SetupRouting(e *echo.Echo, h *Handlers) {
	oauth := h.OAuth2

	e.Validator = validator.New()
	e.HTTPErrorHandler = CustomHTTPErrorHandler
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:8080"},
		AllowCredentials: true,
	}))

	api := e.Group("/api/1.0")
	api.Use(h.UserAuthenticate(oauth))
	apiNoAuth := e.Group("/api/1.0")

	// middleware preparation
	requires := AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := RequestBodyLengthLimit

	// login/logout
	apiNoAuth.POST("/login", h.PostLogin)
	apiNoAuth.POST("/logout", PostLogout)

	// Tag: public
	apiNoAuth.GET("/public/icon/:username", h.GetPublicUserIcon)

	// Tag: channel
	api.GET("/channels", h.GetChannels, requires(permission.GetChannel))
	api.POST("/channels", h.PostChannels, requires(permission.CreateChannel))
	api.GET("/channels/:channelID", h.GetChannelByChannelID, requires(permission.GetChannel))
	api.PATCH("/channels/:channelID", h.PatchChannelByChannelID, requires(permission.EditChannel))
	api.DELETE("/channels/:channelID", h.DeleteChannelByChannelID, requires(permission.DeleteChannel))
	api.PUT("/channels/:channelID/parent", h.PutChannelParent, requires(permission.ChangeParentChannel))
	api.POST("/channels/:channelID/children", h.PostChannelChildren, requires(permission.CreateChannel))

	// Tag: Topic
	api.GET("/channels/:channelID/topic", h.GetTopic, requires(permission.GetTopic))
	api.PUT("/channels/:channelID/topic", h.PutTopic, requires(permission.EditTopic))

	// Tag: messages
	api.GET("/messages/:messageID", h.GetMessageByID, requires(permission.GetMessage))
	api.PUT("/messages/:messageID", h.PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
	api.DELETE("/messages/:messageID", h.DeleteMessageByID, requires(permission.DeleteMessage))
	api.POST("/messages/:messageID/report", h.PostMessageReport, requires(permission.ReportMessage))
	api.GET("/reports", h.GetMessageReports, requires(permission.GetMessageReports))
	api.GET("/channels/:channelID/messages", h.GetMessagesByChannelID, requires(permission.GetMessage))
	api.POST("/channels/:channelID/messages", h.PostMessage, bodyLimit(100), requires(permission.PostMessage))
	api.GET("/users/:userID/messages", h.GetDirectMessages, requires(permission.GetMessage))
	api.POST("/users/:userID/messages", h.PostDirectMessage, bodyLimit(100), requires(permission.PostMessage))

	// Tag: users
	api.GET("/users", h.GetUsers, requires(permission.GetUser))
	api.POST("/users", h.PostUsers, requires(permission.RegisterUser))
	api.GET("/users/me", h.GetMe, requires(permission.GetMe))
	api.PATCH("/users/me", h.PatchMe, requires(permission.EditMe))
	api.PUT("/users/me/password", h.PutPassword, requires(permission.ChangeMyPassword))
	api.GET("/users/me/icon", h.GetMyIcon, requires(permission.DownloadFile))
	api.PUT("/users/me/icon", h.PutMyIcon, requires(permission.ChangeMyIcon))
	api.GET("/users/:userID", h.GetUserByID, requires(permission.GetUser))
	api.GET("/users/:userID/icon", h.GetUserIcon, requires(permission.DownloadFile))

	// Tag: sessions
	api.GET("/users/me/sessions", GetMySessions, requires(permission.GetMySessions))
	api.DELETE("/users/me/sessions", DeleteAllMySessions, requires(permission.DeleteMySessions))
	api.DELETE("/users/me/sessions/:referenceID", DeleteMySession, requires(permission.DeleteMySessions))

	// Tag: clips
	api.GET("/users/me/clips", h.GetClips, requires(permission.GetClip))
	api.POST("/users/me/clips", h.PostClip, requires(permission.CreateClip))
	api.GET("/users/me/clips/:clipID", h.GetClip, requires(permission.GetClip))
	api.DELETE("/users/me/clips/:clipID", h.DeleteClip, requires(permission.DeleteClip))
	api.GET("/users/me/clips/:clipID/folder", h.GetClipsFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PUT("/users/me/clips/:clipID/folder", h.PutClipsFolder, requires(permission.CreateClip))
	api.GET("/users/me/clips/folders", h.GetClipFolders, requires(permission.GetClipFolder))
	api.POST("/users/me/clips/folders", h.PostClipFolder, requires(permission.CreateClipFolder))
	api.GET("/users/me/clips/folders/:folderID", h.GetClipFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PATCH("/users/me/clips/folders/:folderID", h.PatchClipFolder, requires(permission.PatchClipFolder))
	api.DELETE("/users/me/clips/folders/:folderID", h.DeleteClipFolder, requires(permission.DeleteClipFolder))

	// Tag: star
	api.GET("/users/me/stars", h.GetStars, requires(permission.GetStar))
	api.PUT("/users/me/stars/:channelID", h.PutStars, requires(permission.CreateStar))
	api.DELETE("/users/me/stars/:channelID", h.DeleteStars, requires(permission.DeleteStar))

	// Tag: unread
	api.GET("/users/me/unread", h.GetUnread, requires(permission.GetUnread))
	api.DELETE("/users/me/unread/:channelID", h.DeleteUnread, requires(permission.DeleteUnread))

	// Tag: mute
	api.GET("/users/me/mute", h.GetMutedChannelIDs, requires(permission.GetMutedChannels))
	api.POST("/users/me/mute/:channelID", h.PostMutedChannel, requires(permission.MuteChannel))
	api.DELETE("/users/me/mute/:channelID", h.DeleteMutedChannel, requires(permission.UnmuteChannel))

	// Tag: userTag
	api.GET("/users/:userID/tags", h.GetUserTags, requires(permission.GetTag))
	api.POST("/users/:userID/tags", h.PostUserTag, requires(permission.AddTag))
	api.PATCH("/users/:userID/tags/:tagID", h.PatchUserTag, requires(permission.ChangeTagLockState))
	api.DELETE("/users/:userID/tags/:tagID", h.DeleteUserTag, requires(permission.RemoveTag))
	api.GET("/tags", h.GetAllTags, requires(permission.GetTag))
	api.GET("/tags/:tagID", h.GetUsersByTagID, requires(permission.GetTag))
	api.PATCH("/tags/:tagID", h.PatchTag, requires(permission.EditTag))

	// Tag: heartbeat
	api.GET("/heartbeat", h.GetHeartbeat, requires(permission.GetHeartbeat))
	api.POST("/heartbeat", h.PostHeartbeat, requires(permission.PostHeartbeat))

	// Tag: notification
	api.GET("/notification", h.NotificationStream, requires(permission.ConnectNotificationStream))
	api.POST("/notification/device", h.PostDeviceToken, requires(permission.RegisterDevice))
	api.GET("/channels/:channelID/notification", h.GetNotificationStatus, requires(permission.GetNotificationStatus))
	api.PUT("/channels/:channelID/notification", h.PutNotificationStatus, requires(permission.ChangeNotificationStatus))
	api.GET("/users/:userID/notification", h.GetNotificationChannels, requires(permission.GetNotificationStatus))

	// Tag: file
	api.POST("/files", h.PostFile, bodyLimit(30<<10), requires(permission.UploadFile))
	api.GET("/files/:fileID", h.GetFileByID, requires(permission.DownloadFile))
	api.DELETE("/files/:fileID", h.DeleteFileByID, requires(permission.DeleteFile))
	api.GET("/files/:fileID/meta", h.GetMetaDataByFileID, requires(permission.DownloadFile))
	api.GET("/files/:fileID/thumbnail", h.GetThumbnailByID, requires(permission.DownloadFile))

	// Tag: pin
	api.GET("/channels/:channelID/pins", h.GetChannelPin, requires(permission.GetPin))
	api.POST("/pins", h.PostPin, requires(permission.CreatePin))
	api.GET("/pins/:pinID", h.GetPin, requires(permission.GetPin))
	api.DELETE("/pins/:pinID", h.DeletePin, requires(permission.DeletePin))

	// Tag: stamp
	api.GET("/stamps", h.GetStamps, requires(permission.GetStamp))
	api.POST("/stamps", h.PostStamp, requires(permission.CreateStamp))
	api.GET("/stamps/:stampID", h.GetStamp, requires(permission.GetStamp))
	api.PATCH("/stamps/:stampID", h.PatchStamp, requires(permission.EditStamp))
	api.DELETE("/stamps/:stampID", h.DeleteStamp, requires(permission.DeleteStamp))
	api.GET("/messages/:messageID/stamps", h.GetMessageStamps, requires(permission.GetMessageStamp))
	api.POST("/messages/:messageID/stamps/:stampID", h.PostMessageStamp, requires(permission.AddMessageStamp))
	api.DELETE("/messages/:messageID/stamps/:stampID", h.DeleteMessageStamp, requires(permission.RemoveMessageStamp))
	api.GET("/users/me/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))

	// Tag: webhook
	api.GET("/webhooks", h.GetWebhooks, requires(permission.GetWebhook))
	api.POST("/webhooks", h.PostWebhooks, requires(permission.CreateWebhook))
	api.GET("/webhooks/:webhookID", h.GetWebhook, requires(permission.GetWebhook))
	api.PATCH("/webhooks/:webhookID", h.PatchWebhook, requires(permission.EditWebhook))
	api.DELETE("/webhooks/:webhookID", h.DeleteWebhook, requires(permission.DeleteWebhook))
	api.PUT("/webhooks/:webhookID/icon", h.PutWebhookIcon, requires(permission.EditWebhook))
	apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook)
	apiNoAuth.POST("/webhooks/:webhookID/github", h.PostWebhookByGithub)

	// Tag: activity
	api.GET("/activity/latest-messages", h.GetActivityLatestMessages, requires(permission.GetMessage))

	// Tag: user group
	apiGroups := api.Group("/groups")
	{
		apiGroups.GET("", h.GetUserGroups)
		apiGroups.POST("", h.PostUserGroups)

		apiGroupsGid := api.Group("/:groupID", h.ValidateGroupID)
		{
			apiGroupsGid.GET("", h.GetUserGroup)
			apiGroupsGid.PATCH("", h.PatchUserGroup)
			apiGroupsGid.DELETE("", h.DeleteUserGroup)

			apiGroupsGidMembers := api.Group("/members")
			{
				apiGroupsGidMembers.GET("", h.GetUserGroupMembers)
				apiGroupsGidMembers.POST("", h.PostUserGroupMembers)
				apiGroupsGidMembers.DELETE("/:userID", h.DeleteUserGroupMembers)
			}
		}
	}
	api.GET("/users/me/groups", h.GetMyBelongingGroup)
	api.GET("/users/:userID/groups", h.GetUserBelongingGroup)

	if oauth != nil {
		// Tag: bot
		api.GET("/bots", notImplemented, requires(permission.GetBot))
		api.POST("/bots", notImplemented, requires(permission.CreateBot))
		api.GET("/bots/:botID", notImplemented, requires(permission.GetBot))
		api.PATCH("/bots/:botID", notImplemented, requires(permission.EditBot))
		api.DELETE("/bots/:botID", notImplemented, requires(permission.DeleteBot))
		api.PUT("/bots/:botID/icon", notImplemented, requires(permission.EditBot))
		api.POST("/bots/:botID/activation", notImplemented, requires(permission.EditBot))
		api.GET("/bots/:botID/token", notImplemented, requires(permission.GetBotToken))
		api.POST("/bots/:botID/token", notImplemented, requires(permission.ReissueBotToken))
		api.GET("/bots/:botID/code", notImplemented, requires(permission.GetBotInstallCode))
		api.GET("/channels/:channelID/bots", notImplemented, requires(permission.GetBot))
		api.POST("/channels/:channelID/bots", notImplemented, requires(permission.InstallBot))
		api.DELETE("/channels/:channelID/bots/:botID", notImplemented, requires(permission.UninstallBot))

		// Tag: authorization
		apiNoAuth.GET("/oauth2/authorize", oauth.AuthorizationEndpointHandler)
		apiNoAuth.POST("/oauth2/authorize", oauth.AuthorizationEndpointHandler)
		api.POST("/oauth2/authorize/decide", oauth.AuthorizationDecideHandler)
		apiNoAuth.POST("/oauth2/token", oauth.TokenEndpointHandler)
		e.GET("/.well-known/openid-configuration", oauth.DiscoveryHandler)
		e.GET("/publickeys", oauth.PublicKeysHandler)

		// Tag: client
		api.GET("/users/me/tokens", h.GetMyTokens, requires(permission.GetMyTokens))
		api.DELETE("/users/me/tokens/:tokenID", h.DeleteMyToken, requires(permission.RevokeMyToken))
		api.GET("/clients", h.GetClients, requires(permission.GetClients))
		api.POST("/clients", h.PostClients, requires(permission.CreateClient))
		api.GET("/clients/:clientID", h.GetClient, requires(permission.GetClients))
		api.PATCH("/clients/:clientID", h.PatchClient, requires(permission.EditMyClient))
		api.DELETE("/clients/:clientID", h.DeleteClient, requires(permission.DeleteMyClient))
	}

	apiNoAuth.GET("/teapot", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusTeapot, "I'm a teapot")
	})
}

func notImplemented(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented)
}
