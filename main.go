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

	e := echo.New()
	store, err := mysqlstore.NewMySQLStoreFromConnection(engine.DB().DB, "sessions", "/", 60*60*24*14)

	if err != nil {
		panic(err)
	}
	e.Use(session.Middleware(store))
	e.HTTPErrorHandler = router.CustomHTTPErrorHandler

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Tag: channel
	e.GET("/channels", router.GetChannels)
	e.POST("/channels", router.PostChannels)
	e.GET("/channels/:channelID", router.GetChannelsByChannelID)
	e.PUT("/channels/:channelID", router.PutChannelsByChannelID)
	e.DELETE("/channels/:channelID", router.DeleteChannelsByChannelID)

	e.Logger.Fatal(e.Start(":9000"))
}
