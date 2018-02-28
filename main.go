package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/traPtitech/traQ/notification"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router"
)

func main() {
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
	// Tag: channel
	api.GET("/channels", router.GetChannels)
	api.POST("/channels", router.PostChannels)
	api.GET("/channels/:channelID", router.GetChannelsByChannelID)
	api.PUT("/channels/:channelID", router.PutChannelsByChannelID)
	api.DELETE("/channels/:channelID", router.DeleteChannelsByChannelID)

	// Tag: messages
	api.GET("/messages/:messageID", router.GetMessageByID)
	api.PUT("/messages/:messageID", router.PutMessageByID)
	api.DELETE("/messages/:messageID", router.DeleteMessageByID)

	api.GET("/channels/:channelID/messages", router.GetMessagesByChannelID)
	api.POST("/channels/:channelID/messages", router.PostMessage)

	// Tag: users
	api.GET("/users", router.GetUsers)
	api.GET("/users/me", router.GetMe)
	api.GET("/users/:userID", router.GetUserByID)

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

	// Tag: pin
	api.GET("/channels/:channelID/pin", router.GetChannelPin)
	api.POST("/channels/:channelID/pin", router.PostPin)
	api.GET("/pin/:pinID", router.GetPin)
	api.DELETE("/pin/:pinID", router.DeletePin)

	// Serve UI
	e.Static("/static", "./client/dist/static")
	e.File("*", "./client/dist/index.html")

	// init notification
	notification.Start()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
