package model

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"os"
	"testing"
)

var (
	engine *xorm.Engine
)

func TestMain(m *testing.M) {
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

	dbname := "traq-test-model"

	var err error
	engine, err = xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=true", user, pass, host, dbname))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	engine.ShowSQL(false)
	engine.DropTables("sessions", "messages")
	engine.SetMapper(core.GonicMapper{})
	SetXORMEngine(engine)

	err = SyncSchema()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.Exit(code)
}

func beforeTest(t *testing.T) {
	engine.DropTables("sessions", "messages")
	if err := SyncSchema(); err != nil {
		t.Fatal(err)
	}
}
