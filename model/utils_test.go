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
	engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips", "stars", "users_tags", "tags", "devices", "users_subscribe_channels")

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
	engine.DropTables("sessions", "channels", "users_private_channels", "messages", "users", "clips", "stars", "users_tags", "tags", "devices", "users_subscribe_channels")
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

func makeChannelMessages(channelID string) []*Message {
	var messages [10]*Message

	for i := 0; i < 10; i++ {
		tmp := makeMessage()
		messages[i] = tmp
		messages[i].ChannelID = channelID
	}

	return messages[:]
}

func makeUser(userName string) (*User, error) {
	user := &User{
		Name:  userName,
		Email: "hogehoge@gmail.com",
		Icon:  "po",
	}

	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("Failed to setPassword: %v", err)
	}
	if err := user.Create(); err != nil {
		return nil, fmt.Errorf("Failed to user Create: %v", err)
	}

	return user, nil
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
