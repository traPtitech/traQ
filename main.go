package main

import (
	"fmt"
	"net/http"
	"os"

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
	e.POST("/login", router.PostLogin)
	e.POST("/logout", router.PostLogout)

	// Tag: channel
	e.GET("/channels", router.GetChannels, router.GetUserInfo)
	e.POST("/channels", router.PostChannels, router.GetUserInfo)
	e.GET("/channels/:channelID", router.GetChannelsByChannelID, router.GetUserInfo)
	e.PUT("/channels/:channelID", router.PutChannelsByChannelID, router.GetUserInfo)
	e.DELETE("/channels/:channelID", router.DeleteChannelsByChannelID, router.GetUserInfo)

	// Tag: messages
	e.GET("/messages/:messageID", router.GetMessageByID, router.GetUserInfo)
	e.PUT("/messages/:messageID", router.PutMessageByID, router.GetUserInfo)
	e.DELETE("/messages/:messageID", router.DeleteMessageByID, router.GetUserInfo)

	e.GET("/channels/:channelID/messages", router.GetMessagesByChannelID, router.GetUserInfo)
	e.POST("/channels/:channelID/messages", router.PostMessage, router.GetUserInfo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
