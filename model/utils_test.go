package model

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/stretchr/testify/require"
)

var (
	nobodyID = "0ce216f1-4a0d-4011-9f55-d0f79cfb7ca1"
)

func TestMain(m *testing.M) {
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

	port := os.Getenv("MARIADB_PORT")
	if port == "" {
		port = "3306"
	}

	dbname := "traq-test-model"

	engine, err := xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, dbname))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	engine.ShowSQL(false)
	engine.SetMapper(core.GonicMapper{})
	SetXORMEngine(engine)

	if err := SyncSchema(); err != nil {
		panic(err)
	}

	code := m.Run()

	fm := NewDevFileManager()
	os.RemoveAll(fm.GetDir())
	os.Exit(code)
}

// beforeTest : データーベースを初期化して、テストユーザー・テストチャンネルを作成します。
// assert, require, テストユーザー, テストチャンネルを返します。
func beforeTest(t *testing.T) (*assert.Assertions, *require.Assertions, *User, *Channel) {
	require.NoError(t, DropTables())
	require.NoError(t, SyncSchema())

	user := mustMakeUser(t, "testuser")
	return assert.New(t), require.New(t), user, mustMakeChannelDetail(t, user.ID, "testchannel", "", true)
}

func mustMakeChannel(t *testing.T, userID, tail string) *Channel {
	channel := &Channel{}
	channel.CreatorID = userID
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

func mustMakeInvisibleChannel(t *testing.T, channelID, userID string) *UserInvisibleChannel {
	i := &UserInvisibleChannel{}
	i.UserID = userID
	i.ChannelID = channelID
	require.NoError(t, i.Create())
	return i
}

func mustMakeMessage(t *testing.T, userID, channelID string) *Message {
	message := &Message{
		UserID:    userID,
		ChannelID: channelID,
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

func mustMakeFile(t *testing.T, userID string) *File {
	file := &File{
		Name:      "test.txt",
		Size:      90,
		CreatorID: userID,
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
