package main

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/sessions"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/external/storage"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/oauth2/impl"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/utils/validator"
)

func main() {
	// enable pprof http handler
	if len(config.PprofEnabled) > 0 {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Database
	engine, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true", config.DatabaseUserName, config.DatabasePassword, config.DatabaseHostName, config.DatabaseName))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	model.SetGORMEngine(engine)

	if err := model.Sync(); err != nil {
		panic(err)
	}

	sessionStore, err := sessions.NewGORMStore(engine)
	if err != nil {
		panic(err)
	}
	sessions.SetStore(sessionStore)

	// ObjectStorage
	if err := setSwiftFileManagerAsDefault(
		config.OSContainer,
		config.OSUserName,
		config.OSPassword,
		config.OSTenantName, //v2のみ
		config.OSTenantID,   //v2のみ
		config.OSAuthURL,
	); err != nil {
		panic(err)
	}

	// Init Caches
	if err := model.InitCache(); err != nil {
		panic(err)
	}

	// Init Role-Based Access Controller
	r, err := rbac.New(&model.RBACOverrideStore{})
	if err != nil {
		panic(err)
	}
	role.SetRole(r)

	// oauth2 handler
	oauth := &oauth2.Handler{
		Store:                &impl.DefaultStore{},
		AccessTokenExp:       60 * 60 * 24 * 365, //1年
		AuthorizationCodeExp: 60 * 5,             //5分
		IsRefreshEnabled:     false,
		UserAuthenticator: func(id, pw string) (uuid.UUID, error) {
			user, err := model.GetUserByName(id)
			if err != nil {
				switch err {
				case model.ErrNotFound:
					return uuid.Nil, oauth2.ErrUserIDOrPasswordWrong
				default:
					return uuid.Nil, err
				}
			}

			err = model.AuthenticateUser(user, pw)
			switch err {
			case model.ErrUserWrongIDOrPassword, model.ErrUserBotTryLogin:
				err = oauth2.ErrUserIDOrPasswordWrong
			}
			return uuid.FromStringOrNil(user.ID), err
		},
		UserInfoGetter: func(uid uuid.UUID) (oauth2.UserInfo, error) {
			u, err := model.GetUser(uid)
			if err == model.ErrNotFound {
				return nil, oauth2.ErrUserIDOrPasswordWrong
			}
			return u, err
		},
		Issuer: config.TRAQOrigin,
	}
	if public, private := config.RS256PublicKeyFile, config.RS256PrivateKeyFile; private != "" && public != "" {
		err := oauth.LoadKeys(loadKeys(private, public))
		if err != nil {
			panic(err)
		}
	}

	// event handler
	streamer := event.NewSSEStreamer()
	event.AddListener(streamer)
	if len(config.FirebaseServiceAccountJSONFile) > 0 {
		fcm := &event.FCMManager{}
		if err := fcm.Init(); err != nil {
			panic(err)
		}
		event.AddListener(fcm)
	}

	h := router.Handlers{
		Bot:    event.NewBotProcessor(oauth),
		OAuth2: oauth,
	}
	event.AddListener(h.Bot)

	e := echo.New()
	e.Validator = validator.New()
	e.HTTPErrorHandler = router.CustomHTTPErrorHandler
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:8080"},
		AllowCredentials: true,
	}))

	// Serve documents
	e.File("/api/swagger.yaml", "./docs/swagger.yaml")
	e.Static("/api", "./docs/swagger-ui")
	e.Any("/api", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, c.Path()+"/")
	})

	// login/logout
	e.File("/login", "./client/dist/index.html")
	e.POST("/login", router.PostLogin)
	e.POST("/logout", router.PostLogout)

	api := e.Group("/api/1.0")
	api.Use(router.UserAuthenticate(oauth))
	apiNoAuth := e.Group("/api/1.0")

	// access control middleware generator
	requires := router.AccessControlMiddlewareGenerator(r)

	// Tag: channel
	api.GET("/channels", router.GetChannels, requires(permission.GetChannel))
	api.POST("/channels", router.PostChannels, requires(permission.CreateChannel))
	api.GET("/channels/:channelID", router.GetChannelByChannelID, requires(permission.GetChannel))
	api.PATCH("/channels/:channelID", router.PatchChannelByChannelID, requires(permission.EditChannel))
	api.DELETE("/channels/:channelID", router.DeleteChannelByChannelID, requires(permission.DeleteChannel))
	api.PUT("/channels/:channelID/parent", router.PutChannelParent, requires(permission.ChangeParentChannel))

	// Tag: Topic
	api.GET("/channels/:channelID/topic", router.GetTopic, requires(permission.GetTopic))
	api.PUT("/channels/:channelID/topic", router.PutTopic, requires(permission.EditTopic))

	// Tag: messages
	api.GET("/messages/:messageID", router.GetMessageByID, requires(permission.GetMessage))
	api.PUT("/messages/:messageID", router.PutMessageByID, requires(permission.EditMessage))
	api.DELETE("/messages/:messageID", router.DeleteMessageByID, requires(permission.DeleteMessage))
	api.POST("/messages/:messageID/report", router.PostMessageReport, requires(permission.ReportMessage))
	api.GET("/reports", router.GetMessageReports, requires(permission.GetMessageReports))
	api.GET("/channels/:channelID/messages", router.GetMessagesByChannelID, requires(permission.GetMessage))
	api.POST("/channels/:channelID/messages", router.PostMessage, requires(permission.PostMessage))

	// Tag: users
	api.GET("/users", router.GetUsers, requires(permission.GetUser))
	api.POST("/users", router.PostUsers, requires(permission.RegisterUser))
	api.GET("/users/me", router.GetMe, requires(permission.GetMe))
	api.PATCH("/users/me", router.PatchMe, requires(permission.EditMe))
	api.PUT("/users/me/password", router.PutPassword, requires(permission.ChangeMyPassword))
	api.GET("/users/me/icon", router.GetMyIcon, requires(permission.DownloadFile))
	api.PUT("/users/me/icon", router.PutMyIcon, requires(permission.ChangeMyIcon))
	api.GET("/users/:userID", router.GetUserByID, requires(permission.GetUser))
	api.GET("/users/:userID/icon", router.GetUserIcon, requires(permission.DownloadFile))

	// Tag: sessions
	api.GET("/users/me/sessions", router.GetMySessions, requires(permission.GetMySessions))
	api.DELETE("/users/me/sessions", router.DeleteAllMySessions, requires(permission.DeleteMySessions))
	api.DELETE("/users/me/sessions/:referenceID", router.DeleteMySession, requires(permission.DeleteMySessions))

	// Tag: clips
	api.GET("/users/me/clips", router.GetClips, requires(permission.GetClip))
	api.POST("/users/me/clips", router.PostClip, requires(permission.CreateClip))
	api.GET("/users/me/clips/:clipID", router.GetClip, requires(permission.GetClip))
	api.DELETE("/users/me/clips/:clipID", router.DeleteClip, requires(permission.DeleteClip))
	api.GET("/users/me/clips/:clipID/folder", router.GetClipsFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PUT("/users/me/clips/:clipID/folder", router.PutClipsFolder, requires(permission.CreateClip))
	api.GET("/users/me/clips/folders", router.GetClipFolders, requires(permission.GetClipFolder))
	api.POST("/users/me/clips/folders", router.PostClipFolder, requires(permission.CreateClipFolder))
	api.GET("/users/me/clips/folders/:folderID", router.GetClipFolder, requires(permission.GetClip, permission.GetClipFolder))
	api.PATCH("/users/me/clips/folders/:folderID", router.PatchClipFolder, requires(permission.PatchClipFolder))
	api.DELETE("/users/me/clips/folders/:folderID", router.DeleteClipFolder, requires(permission.DeleteClipFolder))

	// Tag: star
	api.GET("/users/me/stars", router.GetStars, requires(permission.GetStar))
	api.PUT("/users/me/stars/:channelID", router.PutStars, requires(permission.CreateStar))
	api.DELETE("/users/me/stars/:channelID", router.DeleteStars, requires(permission.DeleteStar))

	// Tag: unread
	api.GET("/users/me/unread", router.GetUnread, requires(permission.GetUnread))
	api.DELETE("/users/me/unread/:channelID", router.DeleteUnread, requires(permission.DeleteUnread))

	// Tag: mute
	api.GET("/users/me/mute", router.GetMutedChannelIDs, requires(permission.GetMutedChannels))
	api.POST("/users/me/mute/:channelID", router.PostMutedChannel, requires(permission.MuteChannel))
	api.DELETE("/users/me/mute/:channelID", router.DeleteMutedChannel, requires(permission.UnmuteChannel))

	// Tag: userTag
	api.GET("/users/:userID/tags", router.GetUserTags, requires(permission.GetTag))
	api.POST("/users/:userID/tags", router.PostUserTag, requires(permission.AddTag))
	api.PATCH("/users/:userID/tags/:tagID", router.PatchUserTag, requires(permission.ChangeTagLockState))
	api.DELETE("/users/:userID/tags/:tagID", router.DeleteUserTag, requires(permission.RemoveTag))
	api.GET("/tags", router.GetAllTags, requires(permission.GetTag))
	api.GET("/tags/:tagID", router.GetUsersByTagID, requires(permission.GetTag))
	api.PATCH("/tags/:tagID", router.PatchTag, requires(permission.EditTag))

	// Tag: heartbeat
	api.GET("/heartbeat", router.GetHeartbeat, requires(permission.GetHeartbeat))
	api.POST("/heartbeat", router.PostHeartbeat, requires(permission.PostHeartbeat))

	// Tag: notification
	api.GET("/notification", streamer.StreamHandler, requires(permission.ConnectNotificationStream))
	api.POST("/notification/device", router.PostDeviceToken, requires(permission.RegisterDevice))
	api.GET("/channels/:ID/notification", router.GetNotification(router.GetNotificationChannels, router.GetNotificationStatus), requires(permission.GetNotificationStatus))
	api.PUT("/channels/:ID/notification", router.PutNotificationStatus, requires(permission.ChangeNotificationStatus))

	// Tag: file
	api.POST("/files", router.PostFile, requires(permission.UploadFile))
	api.GET("/files/:fileID", router.GetFileByID, requires(permission.DownloadFile))
	api.DELETE("/files/:fileID", router.DeleteFileByID, requires(permission.DeleteFile))
	api.GET("/files/:fileID/meta", router.GetMetaDataByFileID, requires(permission.DownloadFile))
	api.GET("/files/:fileID/thumbnail", router.GetThumbnailByID, requires(permission.DownloadFile))

	// Tag: pin
	api.GET("/channels/:channelID/pin", router.GetChannelPin, requires(permission.GetPin))
	api.POST("/channels/:channelID/pin", router.PostPin, requires(permission.CreatePin))
	api.GET("/pin/:pinID", router.GetPin, requires(permission.GetPin))
	api.DELETE("/pin/:pinID", router.DeletePin, requires(permission.DeletePin))

	// Tag: stamp
	api.GET("/stamps", router.GetStamps, requires(permission.GetStamp))
	api.POST("/stamps", router.PostStamp, requires(permission.CreateStamp))
	api.GET("/stamps/:stampID", router.GetStamp, requires(permission.GetStamp))
	api.PATCH("/stamps/:stampID", router.PatchStamp, requires(permission.EditStamp))
	api.DELETE("/stamps/:stampID", router.DeleteStamp, requires(permission.DeleteStamp))
	api.GET("/messages/:messageID/stamps", router.GetMessageStamps, requires(permission.GetMessageStamp))
	api.POST("/messages/:messageID/stamps/:stampID", router.PostMessageStamp, requires(permission.AddMessageStamp))
	api.DELETE("/messages/:messageID/stamps/:stampID", router.DeleteMessageStamp, requires(permission.RemoveMessageStamp))
	api.GET("/users/me/stamp-history", router.GetMyStampHistory, requires(permission.GetMyStampHistory))

	// Tag: webhook
	router.LoadWebhookTemplate("static/webhook/*.tmpl")
	api.GET("/webhooks", router.GetWebhooks, requires(permission.GetWebhook))
	api.POST("/webhooks", router.PostWebhooks, requires(permission.CreateWebhook))
	api.GET("/webhooks/:webhookID", router.GetWebhook, requires(permission.GetWebhook))
	api.PATCH("/webhooks/:webhookID", router.PatchWebhook, requires(permission.EditWebhook))
	api.DELETE("/webhooks/:webhookID", router.DeleteWebhook, requires(permission.DeleteWebhook))
	api.PUT("/webhooks/:webhookID/icon", router.PutWebhookIcon, requires(permission.EditWebhook))
	apiNoAuth.POST("/webhooks/:webhookID", router.PostWebhook)
	apiNoAuth.POST("/webhooks/:webhookID/github", router.PostWebhookByGithub)

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

	// Serve UI
	e.File("/sw.js", "./client/dist/sw.js")
	e.File("/firebase-messaging-sw.js", "./client/dist/static/firebase-messaging-sw.js")
	e.File("/badge.png", "./static/badge.png")
	e.Static("/static", "./client/dist/static")
	e.File("*", "./client/dist/index.html")

	// init heartbeat
	model.OnUserOnlineStateChanged = func(id string, online bool) {
		if online {
			go event.Emit(event.UserOnline, &event.UserEvent{ID: id})
		} else {
			go event.Emit(event.UserOffline, &event.UserEvent{ID: id})
		}
	}
	model.HeartbeatStart()

	go func() {
		if err := e.Start(":" + config.Port); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Error(err)
	}
	sessions.PurgeCache()
}

func setSwiftFileManagerAsDefault(container, userName, apiKey, tenant, tenantID, authURL string) error {
	if container == "" || userName == "" || apiKey == "" || authURL == "" {
		return nil
	}
	m, err := storage.NewSwiftFileManager(container, userName, apiKey, tenant, tenantID, authURL, false) //TODO リダイレクトをオンにする
	if err != nil {
		return err
	}
	model.SetFileManager("", m)
	return nil
}

func loadKeys(private, public string) ([]byte, []byte) {
	prk, err := ioutil.ReadFile(private)
	if err != nil {
		panic(err)
	}
	puk, err := ioutil.ReadFile(public)
	if err != nil {
		panic(err)
	}
	return prk, puk
}
