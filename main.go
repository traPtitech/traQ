package main

import (
	"fmt"
	"github.com/traPtitech/traQ/auth/oauth2"
	"github.com/traPtitech/traQ/auth/oauth2/impl"
	"github.com/traPtitech/traQ/auth/openid"
	"net/http"
	"os"
	"time"

	"github.com/traPtitech/traQ/notification"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/external"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/router"
)

func main() {
	time.Local = time.UTC

	user := os.Getenv("MARIADB_USERNAME")
	if user == "" {
		user = "root"
	}

	pass := os.Getenv("MARIADB_PASSWORD")
	if pass == "" {
		pass = "password"
	}

	host := os.Getenv("MARIADB_HOSTNAME")
	if host == "" {
		host = "127.0.0.1"
	}

	dbname := os.Getenv("MARIADB_DATABASE")
	if dbname == "" {
		dbname = "traq"
	}

	engine, err := xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=true", user, pass, host, dbname))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	engine.SetMapper(core.GonicMapper{})
	model.SetXORMEngine(engine)

	if err := model.SyncSchema(); err != nil {
		panic(err)
	}

	store, err := mysqlstore.NewMySQLStoreFromConnection(engine.DB().DB, "sessions", "/", 60*60*24*14, []byte("secret"))
	if err != nil {
		panic(err)
	}

	// ObjectStorage
	if err := setSwiftFileManagerAsDefault(
		os.Getenv("OS_CONTAINER"),
		os.Getenv("OS_USERNAME"),
		os.Getenv("OS_PASSWORD"),
		os.Getenv("OS_TENANT_NAME"), //v2のみ
		os.Getenv("OS_TENANT_ID"),   //v2のみ
		os.Getenv("OS_AUTH_URL"),
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
		AccessTokenExp:       60 * 60 * 24 * 365 * 100, //100年
		AuthorizationCodeExp: 60 * 5,                   //5分
		IsRefreshEnabled:     false,
	}

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:8080"},
		AllowCredentials: true,
	}))
	e.Use(session.Middleware(store))
	e.HTTPErrorHandler = router.CustomHTTPErrorHandler

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
	api.GET("/channels/:channelID", router.GetChannelsByChannelID, requires(permission.GetChannel))
	api.PUT("/channels/:channelID", router.PutChannelsByChannelID, requires(permission.EditChannel))
	api.DELETE("/channels/:channelID", router.DeleteChannelsByChannelID, requires(permission.DeleteChannel))

	// Tag: Topic
	api.GET("/channels/:channelID/topic", router.GetTopic, requires(permission.GetTopic))
	api.PUT("/channels/:channelID/topic", router.PutTopic, requires(permission.EditTopic))

	// Tag: messages
	api.GET("/messages/:messageID", router.GetMessageByID, requires(permission.GetMessage))
	api.PUT("/messages/:messageID", router.PutMessageByID, requires(permission.EditMessage))
	api.DELETE("/messages/:messageID", router.DeleteMessageByID, requires(permission.DeleteMessage))
	api.GET("/channels/:channelID/messages", router.GetMessagesByChannelID, requires(permission.GetMessage))
	api.POST("/channels/:channelID/messages", router.PostMessage, requires(permission.PostMessage))

	// Tag: users
	api.GET("/users", router.GetUsers, requires(permission.GetUser))
	api.GET("/users/me", router.GetMe, requires(permission.GetMe))
	api.PATCH("/users/me", router.PatchMe, requires(permission.EditMe))
	api.GET("/users/me/icon", router.GetMyIcon, requires(permission.DownloadFile))
	api.PUT("/users/me/icon", router.PutMyIcon, requires(permission.ChangeMyIcon))
	api.GET("/users/:userID", router.GetUserByID, requires(permission.GetUser))
	api.GET("/users/:userID/icon", router.GetUserIcon, requires(permission.DownloadFile))
	api.POST("/users", router.PostUsers, requires(permission.RegisterUser))

	// Tag: clips
	api.GET("/users/me/clips", router.GetClips, requires(permission.GetClip))
	api.POST("/users/me/clips", router.PostClips, requires(permission.CreateClip))
	api.DELETE("/users/me/clips", router.DeleteClips, requires(permission.DeleteClip))

	// Tag: star
	api.GET("/users/me/stars", router.GetStars, requires(permission.GetStar))
	api.POST("/users/me/stars", router.PostStars, requires(permission.CreateStar))
	api.DELETE("/users/me/stars", router.DeleteStars, requires(permission.DeleteStar))

	// Tag: unread
	api.GET("/users/me/unread", router.GetUnread, requires(permission.GetUnread))
	api.DELETE("/users/me/unread", router.DeleteUnread, requires(permission.DeleteUnread))

	api.GET("/users/:userID/tags", router.GetUserTags, requires(permission.GetTag))
	api.POST("/users/:userID/tags", router.PostUserTag, requires(permission.AddTag))
	api.PUT("/users/:userID/tags/:tagID", router.PutUserTag, requires(permission.ChangeTagLockState))
	api.DELETE("/users/:userID/tags/:tagID", router.DeleteUserTag, requires(permission.RemoveTag))
	api.GET("/tags", router.GetAllTags, requires(permission.GetTag))
	api.GET("/tags/:tagID", router.GetUsersByTagID, requires(permission.GetTag))

	// Tag: heartbeat
	api.GET("/heartbeat", router.GetHeartbeat, requires(permission.GetHeartbeat))
	api.POST("/heartbeat", router.PostHeartbeat, requires(permission.PostHeartbeat))

	// Tag: notification
	api.GET("/notification", router.GetNotificationStream, requires(permission.ConnectNotificationStream))
	api.POST("/notification/device", router.PostDeviceToken, requires(permission.RegisterDevice))
	api.GET("/channels/:channelID/notification", router.GetNotificationStatus, requires(permission.GetNotificationStatus))
	api.PUT("/channels/:channelID/notification", router.PutNotificationStatus, requires(permission.ChangeNotificationStatus))

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

	//Tag: visibility
	api.GET("users/me/channels/visibility", router.GetChannelsVisibility, requires(permission.GetChannelVisibility))
	api.PUT("users/me/channels/visibility", router.PutChannelsVisibility, requires(permission.ChangeChannelVisibility))

	// Tag: webhook
	api.GET("/webhooks", router.GetWebhooks, requires(permission.GetWebhook))
	api.POST("/webhooks", router.PostWebhooks, requires(permission.CreateWebhook))
	api.GET("/webhooks/:webhookID", router.GetWebhook, requires(permission.GetWebhook))
	api.PATCH("/webhooks/:webhookID", router.PatchWebhook, requires(permission.EditWebhook))
	api.DELETE("/webhooks/:webhookID", router.DeleteWebhook, requires(permission.DeleteWebhook))
	apiNoAuth.POST("/webhooks/:webhookID", router.PostWebhook)
	apiNoAuth.POST("/webhooks/:webhookID/github", router.PostWebhookByGithub)

	// Tag: authorization
	apiNoAuth.GET("/oauth2/authorize", oauth.AuthorizationEndpointHandler)
	apiNoAuth.POST("/oauth2/authorize", oauth.AuthorizationEndpointHandler)
	api.POST("/oauth2/authorize/decide", oauth.AuthorizationDecideHandler)
	apiNoAuth.POST("/oauth2/token", oauth.TokenEndpointHandler)
	e.GET("/.well-known/openid-configuration", openid.DiscoveryHandler)
	e.GET("/publickeys", openid.PublicKeysHandler)

	// Tag: client
	oah := &router.OAuth2APIHandler{Store: oauth}
	api.GET("/users/me/tokens", oah.GetMyTokens, requires(permission.GetMyTokens))
	api.DELETE("/users/me/tokens/:tokenID", oah.DeleteMyToken, requires(permission.RevokeMyToken))
	api.GET("/clients", oah.GetClients, requires(permission.GetClients))
	api.POST("/clients", oah.PostClients, requires(permission.CreateClient))
	api.GET("/clients/:clientID", oah.GetClient, requires(permission.GetClients))
	api.PATCH("/clients/:clientID", oah.PatchClient, requires(permission.EditMyClient))
	api.DELETE("/clients/:clientID", oah.DeleteClient, requires(permission.DeleteMyClient))

	// Serve UI
	e.File("/sw.js", "./client/dist/sw.js")
	e.Static("/static", "./client/dist/static")
	e.File("*", "./client/dist/index.html")

	// init notification
	notification.Start()

	// init heartbeat
	model.HeartbeatStart()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	e.Logger.Fatal(e.Start(":" + port))
}

func setSwiftFileManagerAsDefault(container, userName, apiKey, tenant, tenantID, authURL string) error {
	if container == "" || userName == "" || apiKey == "" || authURL == "" {
		return nil
	}
	m, err := external.NewSwiftFileManager(container, userName, apiKey, tenant, tenantID, authURL, false) //TODO リダイレクトをオンにする
	if err != nil {
		return err
	}
	model.SetFileManager("", m)
	return nil
}
