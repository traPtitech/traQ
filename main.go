package main

import (
	"net/http"

	"github.com/labstack/echo"
<<<<<<< 801dbd68dee235d0e48929cc22932b3af478a6e1
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
=======
>>>>>>> wip
	"github.com/traPtitech/traQ/model"
)

func main() {
	if err := model.EstablishConnection(); err != nil {
		panic(err)
	}
<<<<<<< 801dbd68dee235d0e48929cc22932b3af478a6e1
	defer model.Close()
=======
>>>>>>> wip

	if err := model.SyncSchema(); err != nil {
		panic(err)
	}

	e := echo.New()
	store, err := mysqlstore.NewMySQLStoreFromConnection(model.GetSQLDB(), "sessions", "/", 60*60*24*14)

	if err != nil {
		panic(err)
	}
	e.Use(session.Middleware(store))

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(":9000"))
}
