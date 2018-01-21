package model

import (
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

var (
	testUserID    = "403807a5-cae6-453e-8a09-fc75d5b4ca91"
	privateUserID = "8ad765ec-426b-49c1-b4ae-f8af58af9a55"
	engine        *xorm.Engine
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
	engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips")
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
	engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips")
	if err := SyncSchema(); err != nil {
		t.Fatal(err)
	}
}

func makeChannel(tail string) error {
	channel := &Channel{}
	channel.CreatorID = testUserID
	channel.Name = "Channel-" + tail
	channel.IsPublic = true
	return channel.Create()
}

func makeChannelDetail(creatorID, name, parentID string, isPublic bool) (*Channel, error) {
	channel := &Channel{}
	channel.CreatorID = creatorID
	channel.Name = name
	channel.ParentID = parentID
	channel.IsPublic = isPublic
	err := channel.Create()
	return channel, err
}

func makeMessage() *Message {
	message := &Message{
		UserID:    testUserID,
		ChannelID: CreateUUID(),
		Text:      "popopo",
	}
	message.Create()
	return message
}
