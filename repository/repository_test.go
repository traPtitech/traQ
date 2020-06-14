package repository

import (
	"bytes"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

const (
	dbPrefix = "traq-test-repo-"
	common   = "common"
	common2  = "common2"
	common3  = "common3"
	ex1      = "ex1"
	ex2      = "ex2"
	ex3      = "ex3"
	rand     = "random"
)

var (
	repositories = map[string]*GormRepository{}
)

func TestMain(m *testing.M) {
	user := getEnvOrDefault("MARIADB_USERNAME", "root")
	pass := getEnvOrDefault("MARIADB_PASSWORD", "password")
	host := getEnvOrDefault("MARIADB_HOSTNAME", "127.0.0.1")
	port := getEnvOrDefault("MARIADB_PORT", "3306")
	dbs := []string{
		common,
		ex1,
		ex2,
		ex3,
		common2,
		common3,
	}
	if err := migration.CreateDatabasesIfNotExists("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true", user, pass, host, port), dbPrefix, dbs...); err != nil {
		panic(err)
	}

	for _, key := range dbs {
		db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("%s%s", dbPrefix, key)))
		if err != nil {
			panic(err)
		}
		db.DB().SetMaxOpenConns(20)
		if err := migration.DropAll(db); err != nil {
			panic(err)
		}

		repo, err := NewGormRepository(db, storage.NewInMemoryFileStorage(), hub.New(), zap.NewNop())
		if err != nil {
			panic(err)
		}
		if _, err := repo.Sync(); err != nil {
			panic(err)
		}

		repositories[key] = repo.(*GormRepository)
	}

	// Execute tests
	code := m.Run()

	for _, v := range repositories {
		_ = v.db.Close()
		v.hub.Close()
	}
	os.Exit(code)
}

func setup(t *testing.T, repo string) (Repository, *assert.Assertions, *require.Assertions) {
	t.Helper()
	r, ok := repositories[repo]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	return r, assert, require
}

func setupWithUserAndChannel(t *testing.T, repo string) (Repository, *assert.Assertions, *require.Assertions, model.UserInfo, *model.Channel) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeUser(t, r, rand), mustMakeChannel(t, r, rand)
}

func setupWithUser(t *testing.T, repo string) (Repository, *assert.Assertions, *require.Assertions, model.UserInfo) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeUser(t, r, rand)
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}

func getDB(repo Repository) *gorm.DB {
	return repo.(*GormRepository).db
}

func assertAndRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func mustMakeChannel(t *testing.T, repo Repository, name string) *model.Channel {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	ch, err := repo.CreateChannel(model.Channel{
		Name:      name,
		IsForced:  false,
		IsVisible: true,
	}, nil, false)
	require.NoError(t, err)
	return ch
}

func mustMakeMessage(t *testing.T, repo Repository, userID, channelID uuid.UUID) *model.Message {
	t.Helper()
	m, err := repo.CreateMessage(userID, channelID, "popopo")
	require.NoError(t, err)
	return m
}

func mustMakeMessageUnread(t *testing.T, repo Repository, userID, messageID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.SetMessageUnread(userID, messageID, false))
}

func mustMakeUser(t *testing.T, repo Repository, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	// パスワード無し・アイコンファイルは実際には存在しないことに注意
	u, err := repo.CreateUser(CreateUserArgs{Name: userName, Role: role.User, IconFileID: optional.UUIDFrom(uuid.Must(uuid.NewV4()))})
	require.NoError(t, err)
	return u
}

func mustMakeFile(t *testing.T, repo Repository) model.File {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := repo.SaveFile(SaveFileArgs{
		FileName: "test.txt",
		FileSize: int64(buf.Len()),
		FileType: model.FileTypeUserFile,
		Src:      buf,
	})
	require.NoError(t, err)
	return f
}

func mustMakeTag(t *testing.T, repo Repository, name string) *model.Tag {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	tag, err := repo.GetOrCreateTag(name)
	require.NoError(t, err)
	return tag
}

func mustAddTagToUser(t *testing.T, repo Repository, userID, tagID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserTag(userID, tagID))
}

func mustMakePin(t *testing.T, repo Repository, messageID, userID uuid.UUID) {
	t.Helper()
	_, err := repo.PinMessage(messageID, userID)
	require.NoError(t, err)
}

func mustMakeUserGroup(t *testing.T, repo Repository, name string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	g, err := repo.CreateUserGroup(name, "", "", adminID)
	require.NoError(t, err)
	return g
}

func mustAddUserToGroup(t *testing.T, repo Repository, userID, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserToGroup(userID, groupID, ""))
}

func mustMakeStamp(t *testing.T, repo Repository, name string, userID uuid.UUID) *model.Stamp {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	fid, err := GenerateIconFile(repo, name)
	require.NoError(t, err)
	s, err := repo.CreateStamp(CreateStampArgs{Name: name, FileID: fid, CreatorID: userID})
	require.NoError(t, err)
	return s
}

func mustAddMessageStamp(t *testing.T, repo Repository, messageID, stampID, userID uuid.UUID) {
	t.Helper()
	_, err := repo.AddStampToMessage(messageID, stampID, userID, 1)
	require.NoError(t, err)
}

func mustMakeStampPalette(t *testing.T, repo Repository, name, description string, stamps []uuid.UUID, userID uuid.UUID) *model.StampPalette {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	if description == rand {
		description = random.AlphaNumeric(100)
	}
	sp, err := repo.CreateStampPalette(name, description, stamps, userID)
	require.NoError(t, err)
	return sp
}

func mustMakeWebhook(t *testing.T, repo Repository, name string, channelID, creatorID uuid.UUID, secret string) model.Webhook {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	w, err := repo.CreateWebhook(name, "", channelID, creatorID, secret)
	require.NoError(t, err)
	return w
}

func mustChangeChannelSubscription(t *testing.T, repo Repository, channelID, userID uuid.UUID) {
	t.Helper()
	_, _, err := repo.ChangeChannelSubscription(channelID, ChangeChannelSubscriptionArgs{Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{userID: model.ChannelSubscribeLevelMarkAndNotify}})
	require.NoError(t, err)
}

func mustMakeClipFolder(t *testing.T, repo Repository, userID uuid.UUID, name, description string) *model.ClipFolder {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	if description == rand {
		description = random.AlphaNumeric(100)
	}
	cf, err := repo.CreateClipFolder(userID, name, description)
	require.NoError(t, err)
	return cf
}

func mustMakeClipFolderMessage(t *testing.T, repo Repository, folderID, messageID uuid.UUID) *model.ClipFolderMessage {
	t.Helper()
	cfm, err := repo.AddClipFolderMessage(folderID, messageID)
	require.NoError(t, err)
	return cfm
}

func count(t *testing.T, where *gorm.DB) int {
	t.Helper()
	c := 0
	require.NoError(t, where.Count(&c).Error)
	return c
}
