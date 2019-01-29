package router

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/traPtitech/traQ/event"
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
	api.Use(UserAuthenticate(oauth))
	apiNoAuth := e.Group("/api/1.0")

	// middleware preparation
	requires := AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := RequestBodyLengthLimit

	streamer := event.NewSSEStreamer()
	event.AddListener(streamer)

	// login/logout
	apiNoAuth.POST("/login", PostLogin)
	apiNoAuth.POST("/logout", PostLogout)

	// Tag: public
	apiNoAuth.GET("/public/icon/:username", GetPublicUserIcon)

	// Tag: channel
	api.GET("/channels", GetChannels, requires(permission.GetChannel))
	api.POST("/channels", PostChannels, requires(permission.CreateChannel))
	api.GET("/channels/:channelID", GetChannelByChannelID, requires(permission.GetChannel))
	api.PATCH("/channels/:channelID", PatchChannelByChannelID, requires(permission.EditChannel))
	api.DELETE("/channels/:channelID", DeleteChannelByChannelID, requires(permission.DeleteChannel))
	api.PUT("/channels/:channelID/parent", PutChannelParent, requires(permission.ChangeParentChannel))
	api.POST("/channels/:channelID/children", PostChannelChildren, requires(permission.CreateChannel))

	// Tag: Topic
	api.GET("/channels/:channelID/topic", GetTopic, requires(permission.GetTopic))
	api.PUT("/channels/:channelID/topic", PutTopic, requires(permission.EditTopic))

	// Tag: messages
	api.GET("/messages/:messageID", GetMessageByID, requires(permission.GetMessage))
	api.PUT("/messages/:messageID", PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
	api.DELETE("/messages/:messageID", DeleteMessageByID, requires(permission.DeleteMessage))
	api.POST("/messages/:messageID/report", PostMessageReport, requires(permission.ReportMessage))
	api.GET("/reports", GetMessageReports, requires(permission.GetMessageReports))
	api.GET("/channels/:channelID/messages", GetMessagesByChannelID, requires(permission.GetMessage))
	api.POST("/channels/:channelID/messages", PostMessage, bodyLimit(100), requires(permission.PostMessage))
	api.GET("/users/:userID/messages", GetDirectMessages, requires(permission.GetMessage))
	api.POST("/users/:userID/messages", PostDirectMessage, bodyLimit(100), requires(permission.PostMessage))

	// Tag: users
	api.GET("/users", GetUsers, requires(permission.GetUser))
	api.POST("/users", PostUsers, requires(permission.RegisterUser))
	api.GET("/users/me", GetMe, requires(permission.GetMe))
	api.PATCH("/users/me", PatchMe, requires(permission.EditMe))
	api.PUT("/users/me/password", PutPassword, requires(permission.ChangeMyPassword))
	api.GET("/users/me/icon", GetMyIcon, requires(permission.DownloadFile))
	api.PUT("/users/me/icon", PutMyIcon, requires(permission.ChangeMyIcon))
	api.GET("/users/:userID", GetUserByID, requires(permission.GetUser))
	api.GET("/users/:userID/icon", GetUserIcon, requires(permission.DownloadFile))

	// Tag: sessions
	api.GET("/users/me/sessions", GetMySessions, requires(permission.GetMySessions))
	api.DELETE("/users/me/sessions", DeleteAllMySessions, requires(permission.DeleteMySessions))
	api.DELETE("/users/me/sessions/:referenceID", DeleteMySession, requires(permission.DeleteMySessions))

	// Tag: clips
	api.GET("/users/me/clips", GetClips, requires(permission.GetClip))
	api.POST("/users/me/clips", PostClip, requires(permission.CreateClip))
	api.GET("/users/me/clips/:clipID", GetClip, requires(permission.GetClip))
	api.DELETE("/users/me/clips/:clipID", DeleteClip, requires(permission.DeleteClip))
	api.GET("/users/me/clips/:clipID/folder", GetClipsFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PUT("/users/me/clips/:clipID/folder", PutClipsFolder, requires(permission.CreateClip))
	api.GET("/users/me/clips/folders", GetClipFolders, requires(permission.GetClipFolder))
	api.POST("/users/me/clips/folders", PostClipFolder, requires(permission.CreateClipFolder))
	api.GET("/users/me/clips/folders/:folderID", GetClipFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PATCH("/users/me/clips/folders/:folderID", PatchClipFolder, requires(permission.PatchClipFolder))
	api.DELETE("/users/me/clips/folders/:folderID", DeleteClipFolder, requires(permission.DeleteClipFolder))

	// Tag: star
	api.GET("/users/me/stars", GetStars, requires(permission.GetStar))
	api.PUT("/users/me/stars/:channelID", PutStars, requires(permission.CreateStar))
	api.DELETE("/users/me/stars/:channelID", DeleteStars, requires(permission.DeleteStar))

	// Tag: unread
	api.GET("/users/me/unread", GetUnread, requires(permission.GetUnread))
	api.DELETE("/users/me/unread/:channelID", DeleteUnread, requires(permission.DeleteUnread))

	// Tag: mute
	api.GET("/users/me/mute", GetMutedChannelIDs, requires(permission.GetMutedChannels))
	api.POST("/users/me/mute/:channelID", PostMutedChannel, requires(permission.MuteChannel))
	api.DELETE("/users/me/mute/:channelID", DeleteMutedChannel, requires(permission.UnmuteChannel))

	// Tag: userTag
	api.GET("/users/:userID/tags", GetUserTags, requires(permission.GetTag))
	api.POST("/users/:userID/tags", PostUserTag, requires(permission.AddTag))
	api.PATCH("/users/:userID/tags/:tagID", PatchUserTag, requires(permission.ChangeTagLockState))
	api.DELETE("/users/:userID/tags/:tagID", DeleteUserTag, requires(permission.RemoveTag))
	api.GET("/tags", GetAllTags, requires(permission.GetTag))
	api.GET("/tags/:tagID", GetUsersByTagID, requires(permission.GetTag))
	api.PATCH("/tags/:tagID", PatchTag, requires(permission.EditTag))

	// Tag: heartbeat
	api.GET("/heartbeat", GetHeartbeat, requires(permission.GetHeartbeat))
	api.POST("/heartbeat", PostHeartbeat, requires(permission.PostHeartbeat))

	// Tag: notification
	api.GET("/notification", streamer.StreamHandler, requires(permission.ConnectNotificationStream))
	api.POST("/notification/device", PostDeviceToken, requires(permission.RegisterDevice))
	api.GET("/channels/:channelID/notification", GetNotificationStatus, requires(permission.GetNotificationStatus))
	api.PUT("/channels/:channelID/notification", PutNotificationStatus, requires(permission.ChangeNotificationStatus))
	api.GET("/users/:userID/notification", GetNotificationChannels, requires(permission.GetNotificationStatus))

	// Tag: file
	api.POST("/files", PostFile, requires(permission.UploadFile))
	api.GET("/files/:fileID", GetFileByID, requires(permission.DownloadFile))
	api.DELETE("/files/:fileID", DeleteFileByID, requires(permission.DeleteFile))
	api.GET("/files/:fileID/meta", GetMetaDataByFileID, requires(permission.DownloadFile))
	api.GET("/files/:fileID/thumbnail", GetThumbnailByID, requires(permission.DownloadFile))

	// Tag: pin
	api.GET("/channels/:channelID/pins", GetChannelPin, requires(permission.GetPin))
	api.POST("/pins", PostPin, requires(permission.CreatePin))
	api.GET("/pins/:pinID", GetPin, requires(permission.GetPin))
	api.DELETE("/pins/:pinID", DeletePin, requires(permission.DeletePin))

	// Tag: stamp
	api.GET("/stamps", GetStamps, requires(permission.GetStamp))
	api.POST("/stamps", PostStamp, requires(permission.CreateStamp))
	api.GET("/stamps/:stampID", GetStamp, requires(permission.GetStamp))
	api.PATCH("/stamps/:stampID", PatchStamp, requires(permission.EditStamp))
	api.DELETE("/stamps/:stampID", DeleteStamp, requires(permission.DeleteStamp))
	api.GET("/messages/:messageID/stamps", GetMessageStamps, requires(permission.GetMessageStamp))
	api.POST("/messages/:messageID/stamps/:stampID", PostMessageStamp, requires(permission.AddMessageStamp))
	api.DELETE("/messages/:messageID/stamps/:stampID", DeleteMessageStamp, requires(permission.RemoveMessageStamp))
	api.GET("/users/me/stamp-history", GetMyStampHistory, requires(permission.GetMyStampHistory))

	// Tag: webhook
	api.GET("/webhooks", GetWebhooks, requires(permission.GetWebhook))
	api.POST("/webhooks", PostWebhooks, requires(permission.CreateWebhook))
	api.GET("/webhooks/:webhookID", GetWebhook, requires(permission.GetWebhook))
	api.PATCH("/webhooks/:webhookID", PatchWebhook, requires(permission.EditWebhook))
	api.DELETE("/webhooks/:webhookID", DeleteWebhook, requires(permission.DeleteWebhook))
	api.PUT("/webhooks/:webhookID/icon", PutWebhookIcon, requires(permission.EditWebhook))
	apiNoAuth.POST("/webhooks/:webhookID", PostWebhook)
	apiNoAuth.POST("/webhooks/:webhookID/github", PostWebhookByGithub)

	if oauth != nil {
		// Tag: bot
		api.GET("/bots", h.GetBots, requires(permission.GetBot))
		api.POST("/bots", h.PostBots, requires(permission.CreateBot))
		api.GET("/bots/:botID", h.GetBot, requires(permission.GetBot))
		api.PATCH("/bots/:botID", h.PatchBot, requires(permission.EditBot))
		api.DELETE("/bots/:botID", h.DeleteBot, requires(permission.DeleteBot))
		api.PUT("/bots/:botID/icon", h.PutBotIcon, requires(permission.EditBot))
		api.POST("/bots/:botID/activation", h.PostBotActivation, requires(permission.EditBot))
		api.GET("/bots/:botID/token", h.GetBotToken, requires(permission.GetBotToken))
		api.POST("/bots/:botID/token", h.PostBotToken, requires(permission.ReissueBotToken))
		api.GET("/bots/:botID/code", h.GetBotInstallCode, requires(permission.GetBotInstallCode))
		api.GET("/channels/:channelID/bots", h.GetInstalledBots, requires(permission.GetBot))
		api.POST("/channels/:channelID/bots", h.PostInstalledBots, requires(permission.InstallBot))
		api.DELETE("/channels/:channelID/bots/:botID", h.DeleteInstalledBot, requires(permission.UninstallBot))

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
