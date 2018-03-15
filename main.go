package main

import (
	"fmt"
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

	if err := model.InitCache(); err != nil {
		panic(err)
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
	api.Use(router.GetUserInfo)
	apiNoAuth := e.Group("/api/1.0")

	// Tag: channel
	api.GET("/channels", router.GetChannels)
	api.POST("/channels", router.PostChannels)
	api.GET("/channels/:channelID", router.GetChannelsByChannelID)
	api.PUT("/channels/:channelID", router.PutChannelsByChannelID)
	api.DELETE("/channels/:channelID", router.DeleteChannelsByChannelID)

	// Tag: Topic
	api.GET("/channels/:channelID/topic", router.GetTopic)
	api.PUT("/channels/:channelID/topic", router.PutTopic)

	// Tag: messages
	api.GET("/messages/:messageID", router.GetMessageByID)
	api.PUT("/messages/:messageID", router.PutMessageByID)
	api.DELETE("/messages/:messageID", router.DeleteMessageByID)

	api.GET("/channels/:channelID/messages", router.GetMessagesByChannelID)
	api.POST("/channels/:channelID/messages", router.PostMessage)

	// Tag: users
	api.GET("/users", router.GetUsers)
	api.GET("/users/me", router.GetMe)
	api.PATCH("/users/me", router.PatchMe)
	api.GET("/users/me/icon", router.GetMyIcon)
	api.PUT("/users/me/icon", router.PutMyIcon)
	api.GET("/users/:userID", router.GetUserByID)
	api.GET("/users/:userID/icon", router.GetUserIcon)
	api.POST("/users", router.PostUsers)

	// Tag: clips
	api.GET("/users/me/clips", router.GetClips)
	api.POST("/users/me/clips", router.PostClips)
	api.DELETE("/users/me/clips", router.DeleteClips)

	// Tag: star
	api.GET("/users/me/stars", router.GetStars)
	api.POST("/users/me/stars", router.PostStars)
	api.DELETE("/users/me/stars", router.DeleteStars)

	// Tag: unread
	api.GET("/users/me/unread", router.GetUnread, router.GetUserInfo)
	api.DELETE("/users/me/unread", router.DeleteUnread, router.GetUserInfo)

	// Tag: userTag
	api.GET("/users/:userID/tags", router.GetUserTags)
	api.POST("/users/:userID/tags", router.PostUserTag)
	api.PUT("/users/:userID/tags/:tagID", router.PutUserTag)
	api.DELETE("/users/:userID/tags/:tagID", router.DeleteUserTag)
	api.GET("/tags", router.GetAllTags)
	api.GET("/tags/{tagID}", router.GetUsersByTagID)

	// Tag: heartbeat
	api.GET("/heartbeat", router.GetHeartbeat)
	api.POST("/heartbeat", router.PostHeartbeat)

	// Tag: notification
	api.GET("/notification", router.GetNotificationStream)
	api.POST("/notification/device", router.PostDeviceToken)
	api.GET("/channels/:channelID/notification", router.GetNotificationStatus)
	api.PUT("/channels/:channelID/notification", router.PutNotificationStatus)

	// Tag: file
	api.POST("/files", router.PostFile)
	api.GET("/files/:fileID", router.GetFileByID)
	api.DELETE("/files/:fileID", router.DeleteFileByID)
	api.GET("/files/:fileID/meta", router.GetMetaDataByFileID)
	api.GET("/files/:fileID/thumbnail", router.GetThumbnailByID)

	// Tag: pin
	api.GET("/channels/:channelID/pin", router.GetChannelPin)
	api.POST("/channels/:channelID/pin", router.PostPin)
	api.GET("/pin/:pinID", router.GetPin)
	api.DELETE("/pin/:pinID", router.DeletePin)

	// Tag: stamp
	api.GET("/stamps", router.GetStamps)
	api.POST("/stamps", router.PostStamp)
	api.GET("/stamps/:stampID", router.GetStamp)
	api.PATCH("/stamps/:stampID", router.PatchStamp)
	api.DELETE("/stamps/:stampID", router.DeleteStamp)
	api.GET("/messages/:messageID/stamps", router.GetMessageStamps)
	api.POST("/messages/:messageID/stamps/:stampID", router.PostMessageStamp)
	api.DELETE("/messages/:messageID/stamps/:stampID", router.DeleteMessageStamp)

	//Tag: visibility
	api.GET("users/me/channels/visibility", router.GetChannelsVisibility)
	api.PUT("users/me/channels/visibility", router.PutChannelsVisibility)

	// Tag: webhook
	api.GET("/webhooks", router.GetWebhooks)
	api.POST("/webhooks", router.PostWebhooks)
	api.GET("/webhooks/:webhookID", router.GetWebhook)
	api.PATCH("/webhooks/:webhookID", router.PatchWebhook)
	api.DELETE("/webhooks/:webhookID", router.DeleteWebhook)
	apiNoAuth.POST("/webhooks/:webhookID", router.PostWebhook)
	apiNoAuth.POST("/webhooks/:webhookID/github", router.PostWebhookByGithub)

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
