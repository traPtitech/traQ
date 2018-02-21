package model

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/stretchr/testify/require"
)

var (
	testUserID    = "403807a5-cae6-453e-8a09-fc75d5b4ca91"
	nobodyID      = "0ce216f1-4a0d-4011-9f55-d0f79cfb7ca1"
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
	engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips", "stars", "users_tags", "tags", "unreads", "devices", "users_subscribe_channels", "files")

	engine.SetMapper(core.GonicMapper{})
	SetXORMEngine(engine)

	err = SyncSchema()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.RemoveAll(dirName)
	os.Exit(code)
}

func beforeTest(t *testing.T) {
	require.NoError(t, engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips", "stars", "users_tags", "tags", "unreads", "devices", "users_subscribe_channels", "files"))
	require.NoError(t, SyncSchema())
}

func mustMakeChannel(t *testing.T, tail string) *Channel {
	channel := &Channel{}
	channel.CreatorID = testUserID
	channel.Name = "Channel-" + tail
	channel.IsPublic = true
	require.NoError(t, channel.Create())
	return channel
}

func mustMakeChannelDetail(t *testing.T, creatorID, name, parentID string, isPublic bool) *Channel {
	channel := &Channel{}
	channel.CreatorID = creatorID
	channel.Name = name
	channel.ParentID = parentID
	channel.IsPublic = isPublic
	require.NoError(t, channel.Create())
	return channel
}

func mustMakeMessage(t *testing.T) *Message {
	message := &Message{
		UserID:    testUserID,
		ChannelID: CreateUUID(),
		Text:      "popopo",
	}
	require.NoError(t, message.Create())
	return message
}

func mustMakeMessageUnread(t *testing.T, userID, messageID string) *Unread {
	unread := &Unread{
		UserID:    userID,
		MessageID: messageID,
	}
	require.NoError(t, unread.Create())
	return unread
}

func mustMakeUser(t *testing.T, userName string) *User {
	user := &User{
		Name:  userName,
		Email: "hogehoge@gmail.com",
		Icon:  "po",
	}
	require.NoError(t, user.SetPassword(password))
	require.NoError(t, user.Create())
	return user
}

func mustMakeFile(t *testing.T) *File {
	file := &File{
		Name:      "test.txt",
		Size:      90,
		CreatorID: testUserID,
	}
	require.NoError(t, file.Create(bytes.NewBufferString("test message")))
	return file
}

func checkEmptyField(user *User) error {
	if user.ID == "" {
		return fmt.Errorf("ID is empty")
	}
	if user.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if user.Email == "" {
		return fmt.Errorf("Email is empty")
	}
	if user.Password == "" {
		return fmt.Errorf("Password is empty")
	}
	if user.Salt == "" {
		return fmt.Errorf("Salt is empty")
	}
	if user.Icon == "" {
		return fmt.Errorf("Icon is empty")
	}
	if user.Status == 0 {
		return fmt.Errorf("Status is empty")
	}
	return nil
}
