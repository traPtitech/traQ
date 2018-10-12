package model

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"os"
	"testing"

	"github.com/traPtitech/traQ/external/storage"

	"github.com/stretchr/testify/assert"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/rbac/role"
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

	port := os.Getenv("MARIADB_PORT")
	if port == "" {
		port = "3306"
	}

	dbname := "traq-test-model"
	config.DatabaseName = "traq-test-model"

	db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, dbname))
	if err != nil {
		panic(err)
	}
	defer db.Close()
	SetGORMEngine(db)

	// テストで作成されたfileは全てメモリ上に乗ります。容量注意
	SetFileManager("", storage.NewInMemoryFileManager())

	if _, err := Sync(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// beforeTest : データーベースを初期化して、テストユーザー・テストチャンネルを作成します。
// assert, require, テストユーザー, テストチャンネルを返します。
func beforeTest(t *testing.T) (*assert.Assertions, *require.Assertions, *User, *Channel) {
	require.NoError(t, DropTables())
	_, err := Sync()
	require.NoError(t, err)

	user := mustMakeUser(t, "testuser")
	return assert.New(t), require.New(t), user, mustMakeChannelDetail(t, user.GetUID(), "testchannel", "")
}

func mustMakeChannel(t *testing.T, userID uuid.UUID, tail string) *Channel {
	ch, err := CreatePublicChannel("", "Channel-"+tail, userID)
	require.NoError(t, err)
	return ch
}

func mustMakeChannelDetail(t *testing.T, userID uuid.UUID, name, parentID string) *Channel {
	ch, err := CreatePublicChannel(parentID, name, userID)
	require.NoError(t, err)
	return ch
}

func mustMakePrivateChannel(t *testing.T, name string, members []uuid.UUID) *Channel {
	ch, err := CreatePrivateChannel("", name, members[0], members)
	require.NoError(t, err)
	return ch
}

func mustMakeMessage(t *testing.T, userID, channelID uuid.UUID) *Message {
	m, err := CreateMessage(userID, channelID, "popopo")
	require.NoError(t, err)
	return m
}

func mustMakeMessageUnread(t *testing.T, userID, messageID uuid.UUID) {
	require.NoError(t, SetMessageUnread(userID, messageID))
}

func mustMakeUser(t *testing.T, userName string) *User {
	u, err := CreateUser(userName, userName+"@test.test", "test", role.User)
	require.NoError(t, err)
	return u
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

func mustMakeTag(t *testing.T, name string) *Tag {
	tag, err := CreateTag(name, false, "")
	require.NoError(t, err)
	return tag
}
