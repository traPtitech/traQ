package gorm

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
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
	repositories = map[string]*Repository{}
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
		engine, err := gorm.Open(mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("%s%s", dbPrefix, key)),
		}))
		if err != nil {
			panic(err)
		}
		db, err := engine.DB()
		if err != nil {
			panic(err)
		}
		db.SetMaxOpenConns(20)
		engine.Logger = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
		})
		if err := migration.DropAll(engine); err != nil {
			panic(err)
		}

		repo, _, err := NewGormRepository(engine, hub.New(), zap.NewNop(), true)
		if err != nil {
			panic(err)
		}

		repositories[key] = repo.(*Repository)
	}

	// Execute tests
	code := m.Run()

	for _, v := range repositories {
		db, _ := v.db.DB()
		_ = db.Close()
		v.hub.Close()
	}
	os.Exit(code)
}

func setup(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions) {
	t.Helper()
	r, ok := repositories[repo]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	return r, assert, require
}

func setupWithUserAndChannel(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions, model.UserInfo, *model.Channel) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeUser(t, r, rand), mustMakeChannel(t, r, rand)
}

func setupWithUser(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions, model.UserInfo) {
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

func getDB(repo repository.Repository) *gorm.DB {
	return repo.(*Repository).db
}

func assertAndRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func mustMakeChannel(t *testing.T, repo repository.Repository, name string) *model.Channel {
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

func mustMakeMessage(t *testing.T, repo repository.Repository, userID, channelID uuid.UUID) *model.Message {
	t.Helper()
	m, err := repo.CreateMessage(userID, channelID, "popopo")
	require.NoError(t, err)
	return m
}

func mustMakeMessageUnread(t *testing.T, repo repository.Repository, userID, messageID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.SetMessageUnread(userID, messageID, false))
}

func mustMakeUser(t *testing.T, repo repository.Repository, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	// パスワード無し・アイコンファイルは実際には存在しないことに注意
	u, err := repo.CreateUser(repository.CreateUserArgs{Name: userName, Role: role.User, IconFileID: uuid.Must(uuid.NewV4())})
	require.NoError(t, err)
	return u
}

func mustMakeTag(t *testing.T, repo repository.Repository, name string) *model.Tag {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	tag, err := repo.GetOrCreateTag(name)
	require.NoError(t, err)
	return tag
}

func mustAddTagToUser(t *testing.T, repo repository.Repository, userID, tagID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserTag(userID, tagID))
}

func mustMakePin(t *testing.T, repo repository.Repository, messageID, userID uuid.UUID) {
	t.Helper()
	_, err := repo.PinMessage(messageID, userID)
	require.NoError(t, err)
}

func mustMakeUserGroup(t *testing.T, repo repository.Repository, name string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	icon := mustMakeDummyFile(t, repo)
	g, err := repo.CreateUserGroup(name, "", "", adminID, icon.ID)
	require.NoError(t, err)
	return g
}

func mustAddUserToGroup(t *testing.T, repo repository.Repository, userID, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserToGroup(userID, groupID, ""))
}

func mustMakeDummyFile(t *testing.T, repo repository.Repository) *model.FileMeta {
	t.Helper()
	meta := &model.FileMeta{
		ID:   uuid.Must(uuid.NewV4()),
		Name: "dummy",
		Mime: "application/octet-stream",
		Size: 10,
		Hash: "d41d8cd98f00b204e9800998ecf8427e",
		Type: model.FileTypeUserFile,
		Thumbnails: []model.FileThumbnail{
			{
				Type:   model.ThumbnailTypeImage,
				Mime:   "image/png",
				Width:  100,
				Height: 100,
			},
		},
	}
	err := repo.SaveFileMeta(meta, []*model.FileACLEntry{
		{UserID: uuid.Nil, Allow: true},
	})
	require.NoError(t, err)
	return meta
}

func mustMakeStamp(t *testing.T, repo repository.Repository, name string, userID uuid.UUID) *model.Stamp {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	fid := mustMakeDummyFile(t, repo).ID
	s, err := repo.CreateStamp(repository.CreateStampArgs{Name: name, FileID: fid, CreatorID: userID})
	require.NoError(t, err)
	return s
}

func mustMakeStampAlias(t *testing.T, repo repository.Repository, stampID uuid.UUID, name string, userID uuid.UUID) *model.StampAlias {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	sa, err := repo.CreateStampAlias(repository.CreateStampAliasArgs{StampID: stampID, Name: name, CreatorID: userID})
	require.NoError(t, err)
	return sa
}

func mustAddMessageStamp(t *testing.T, repo repository.Repository, messageID, stampID, userID uuid.UUID) {
	t.Helper()
	_, err := repo.AddStampToMessage(messageID, stampID, userID, 1)
	require.NoError(t, err)
}

func mustMakeStampPalette(t *testing.T, repo repository.Repository, name, description string, stamps []uuid.UUID, userID uuid.UUID) *model.StampPalette {
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

func mustMakeWebhook(t *testing.T, repo repository.Repository, name string, channelID, creatorID uuid.UUID, secret string) model.Webhook {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	w, err := repo.CreateWebhook(name, "", channelID, mustMakeDummyFile(t, repo).ID, creatorID, secret)
	require.NoError(t, err)
	return w
}

func mustChangeChannelSubscription(t *testing.T, repo repository.Repository, channelID, userID uuid.UUID) {
	t.Helper()
	_, _, err := repo.ChangeChannelSubscription(channelID, repository.ChangeChannelSubscriptionArgs{Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{userID: model.ChannelSubscribeLevelMarkAndNotify}})
	require.NoError(t, err)
}

func mustMakeClipFolder(t *testing.T, repo repository.Repository, userID uuid.UUID, name, description string) *model.ClipFolder {
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

func mustMakeClipFolderMessage(t *testing.T, repo repository.Repository, folderID, messageID uuid.UUID) *model.ClipFolderMessage {
	t.Helper()
	cfm, err := repo.AddClipFolderMessage(folderID, messageID)
	require.NoError(t, err)
	return cfm
}

func count(t *testing.T, where *gorm.DB) int {
	t.Helper()
	var c int64
	require.NoError(t, where.Count(&c).Error)
	return int(c)
}
