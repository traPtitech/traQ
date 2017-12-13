package model

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

var db *xorm.Engine

func EstablishConnection() error {
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
		return fmt.Errorf("Failed to communicate with db: %v", err)
	}

	engine.SetMapper(core.GonicMapper{})

	db = engine
	return nil
}

func Close() error {
	err := db.Close()
	if err != nil {
		return fmt.Errorf("Failed to close db: %v", err)
	}
	return nil
}

func GetSQLDB() *sql.DB {
	return db.DB().DB
}

func SyncSchema() error {
	if err := db.Sync(new(Messages)); err != nil {
		return fmt.Errorf("Failed to sync Messages: %v", err)
	}
	return nil
}

func CreateUUID() string {
	return uuid.NewV4().String()
}

func BeforeTest(t *testing.T) {
	err := EstablishConnection()
	if err != nil {
		t.Fatal("Failed to EstablishConnection\n", err)
	}

	db.ShowSQL(false)
	db.DropTables("messages")

	err = SyncSchema()
	if err != nil {
		t.Fatal("Failed to SyncSchema\n", err)
	}
}
