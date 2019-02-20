package impl

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"os"
	"testing"
)

const (
	common = "common"
	ex1    = "ex1"
	ex2    = "ex2"
	random = "random"
)

var (
	repositories = map[string]*RepositoryImpl{}
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
	}

	for _, key := range dbs {
		db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("traq-test-repo-%s", key)))
		if err != nil {
			panic(err)
		}
		db.DB().SetMaxOpenConns(20)
		if err := dropTables(db); err != nil {
			panic(err)
		}

		repo, err := NewRepositoryImpl(db, storage.NewInMemoryFileStorage(), hub.New())
		if err != nil {
			panic(err)
		}
		if _, err := repo.Sync(); err != nil {
			panic(err)
		}

		repositories[key] = repo.(*RepositoryImpl)
	}

	// Execute tests
	code := m.Run()

	for _, v := range repositories {
		_ = v.db.Close()
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

func setupWithUserAndChannel(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions, *model.User, *model.Channel) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeUser(t, r, random), mustMakeChannel(t, r, random)
}

func setupWithChannel(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions, *model.Channel) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeChannel(t, r, random)
}

func setupWithUser(t *testing.T, repo string) (repository.Repository, *assert.Assertions, *require.Assertions, *model.User) {
	t.Helper()
	r, assert, require := setup(t, repo)
	return r, assert, require, mustMakeUser(t, r, random)
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}

func getDB(repo repository.Repository) *gorm.DB {
	return repo.(*RepositoryImpl).db
}

func assertAndRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func dropTables(db *gorm.DB) error {
	for _, v := range model.Tables {
		if err := db.DropTableIfExists(v).Error; err != nil {
			return err
		}
	}
	return nil
}

func mustMakeChannel(t *testing.T, repo repository.Repository, name string) *model.Channel {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	ch, err := repo.CreatePublicChannel(name, uuid.Nil, uuid.Nil)
	require.NoError(t, err)
	return ch
}

func mustMakeChannelDetail(t *testing.T, repo repository.Repository, userID uuid.UUID, name string, parentID uuid.UUID) *model.Channel {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	ch, err := repo.CreatePublicChannel(name, parentID, userID)
	require.NoError(t, err)
	return ch
}

func mustMakePrivateChannel(t *testing.T, repo repository.Repository, name string, members []uuid.UUID) *model.Channel {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	ch, err := repo.CreatePrivateChannel(name, members[0], members)
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
	require.NoError(t, repo.SetMessageUnread(userID, messageID))
}

func mustMakeUser(t *testing.T, repo repository.Repository, userName string) *model.User {
	t.Helper()
	if userName == random {
		userName = utils.RandAlphabetAndNumberString(32)
	}
	u, err := repo.CreateUser(userName, "test", role.User)
	require.NoError(t, err)
	return u
}

func mustMakeFile(t *testing.T, repo repository.Repository, userID uuid.UUID) *model.File {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := repo.SaveFile("test.txt", buf, int64(buf.Len()), "", model.FileTypeUserFile, userID)
	require.NoError(t, err)
	return f
}

func mustMakeTag(t *testing.T, repo repository.Repository, name string) *model.Tag {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	tag, err := repo.CreateTag(name, false, "")
	require.NoError(t, err)
	return tag
}

func mustMakePin(t *testing.T, repo repository.Repository, messageID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	p, err := repo.CreatePin(messageID, userID)
	require.NoError(t, err)
	return p
}

func mustMakeUserGroup(t *testing.T, repo repository.Repository, name string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	g, err := repo.CreateUserGroup(name, "", adminID)
	require.NoError(t, err)
	return g
}

func mustAddUserToGroup(t *testing.T, repo repository.Repository, userID, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserToGroup(userID, groupID))
}

func count(t *testing.T, where *gorm.DB) int {
	t.Helper()
	c := 0
	require.NoError(t, where.Count(&c).Error)
	return c
}
