package v1

import (
	"bytes"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/testutils"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"

	"github.com/stretchr/testify/assert"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/rbac/role"
)

const (
	rand    = "random"
	common1 = "common1"
	common2 = "common2"
	common3 = "common3"
	common4 = "common4"
	common5 = "common5"
	common6 = "common6"
	s1      = "s1"
	s2      = "s2"
	s3      = "s3"
	s4      = "s4"
)

var envs = map[string]*Env{}

func TestMain(m *testing.M) {
	// setup server
	repos := []string{
		common1,
		common2,
		common3,
		common4,
		common5,
		common6,
		s1,
		s2,
		s3,
		s4,
	}
	for _, key := range repos {
		env := &Env{}
		env.Repository = testutils.NewTestRepository()
		env.Hub = hub.New()
		env.SessStore = session.NewMemorySessionStore()
		env.RBAC = testutils.NewTestRBAC()
		env.ChannelManager, _ = channel.InitChannelManager(env.Repository, zap.NewNop())
		env.MessageManager, _ = message.NewMessageManager(env.Repository, env.ChannelManager, zap.NewNop())
		env.ImageProcessor = imaging.NewProcessor(imaging.Config{
			MaxPixels:        1000 * 1000,
			Concurrency:      1,
			ThumbnailMaxSize: image.Pt(360, 480),
			ImageMagickPath:  "",
		})
		env.FileManager, _ = file.InitFileManager(env.Repository, storage.NewInMemoryFileStorage(), env.ImageProcessor, zap.NewNop())

		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap(env.Repository, env.ChannelManager))

		handlers := &Handlers{
			RBAC:           env.RBAC,
			Repo:           env.Repository,
			Hub:            env.Hub,
			Logger:         zap.NewNop(),
			OC:             counter.NewOnlineCounter(env.Hub),
			VM:             viewer.NewManager(env.Hub),
			ChannelManager: env.ChannelManager,
			MessageManager: env.MessageManager,
			FileManager:    env.FileManager,
			SessStore:      env.SessStore,
			Imaging:        env.ImageProcessor,
		}
		handlers.Setup(e.Group("/api"))
		env.Server = httptest.NewServer(e)
		envs[key] = env
	}

	code := m.Run()

	for _, env := range envs {
		env.Server.Close()
		env.Hub.Close()
	}
	os.Exit(code)
}

type Env struct {
	Server         *httptest.Server
	Repository     repository.Repository
	Hub            *hub.Hub
	SessStore      session.Store
	RBAC           rbac.RBAC
	ChannelManager channel.Manager
	MessageManager message.Manager
	FileManager    file.Manager
	ImageProcessor imaging.Processor
}

func setup(t *testing.T, server string) (*Env, *assert.Assertions, *require.Assertions, string, string) {
	t.Helper()
	env, ok := envs[server]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	repo := env.Repository
	testUser := env.mustMakeUser(t, rand)
	adminUser, err := repo.GetUserByName("traq", true)
	require.NoError(err)
	return env, assert, require, env.generateSession(t, testUser.GetID()), env.generateSession(t, adminUser.GetID())
}

func setupWithUsers(t *testing.T, server string) (*Env, *assert.Assertions, *require.Assertions, string, string, model.UserInfo, model.UserInfo) {
	t.Helper()
	env, ok := envs[server]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	repo := env.Repository
	testUser := env.mustMakeUser(t, rand)
	adminUser, err := repo.GetUserByName("traq", true)
	require.NoError(err)
	return env, assert, require, env.generateSession(t, testUser.GetID()), env.generateSession(t, adminUser.GetID()), testUser, adminUser
}

func assertAndRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func (env *Env) generateSession(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	sess, err := env.SessStore.IssueSession(userID, nil)
	require.NoError(t, err)
	return sess.Token()
}

func (env *Env) makeExp(t *testing.T) *httpexpect.Expect {
	t.Helper()
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  env.Server.URL,
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewCurlPrinter(t),
			httpexpect.NewDebugPrinter(t, true),
		},
		Client: &http.Client{
			Jar:     nil, // クッキーは保持しない
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // リダイレクトを自動処理しない
			},
		},
	})
}

func (env *Env) mustMakeChannel(t *testing.T, name string) *model.Channel {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	ch, err := env.ChannelManager.CreatePublicChannel(name, uuid.Nil, uuid.Nil)
	require.NoError(t, err)
	return ch
}

func (env *Env) mustMakeMessage(t *testing.T, userID, channelID uuid.UUID) *model.Message {
	t.Helper()
	m, err := env.Repository.CreateMessage(userID, channelID, "popopo")
	require.NoError(t, err)
	return m
}

func (env *Env) mustMakeMessageUnread(t *testing.T, userID, messageID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.SetMessageUnread(userID, messageID, false))
}

func (env *Env) mustMakeUser(t *testing.T, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	// パスワード無し・アイコンファイルは実際には存在しないことに注意
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Role: role.User, IconFileID: uuid.Must(uuid.NewV4())})
	require.NoError(t, err)
	return u
}

func (env *Env) mustMakeFile(t *testing.T) model.File {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := env.FileManager.Save(file.SaveArgs{
		FileName: "test.txt",
		FileSize: int64(buf.Len()),
		FileType: model.FileTypeUserFile,
		Src:      buf,
	})
	require.NoError(t, err)
	return f
}

func (env *Env) mustMakeTag(t *testing.T, userID uuid.UUID, tagText string) uuid.UUID {
	t.Helper()
	if tagText == rand {
		tagText = random.AlphaNumeric(20)
	}
	tag, err := env.Repository.GetOrCreateTag(tagText)
	require.NoError(t, err)
	require.NoError(t, env.Repository.AddUserTag(userID, tag.ID))
	return tag.ID
}

func (env *Env) mustStarChannel(t *testing.T, userID, channelID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.AddStar(userID, channelID))
}

func (env *Env) mustMakeUserGroup(t *testing.T, name string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	icon := env.mustMakeFile(t)
	g, err := env.Repository.CreateUserGroup(name, "", "", adminID, icon.GetID())
	require.NoError(t, err)
	return g
}

func (env *Env) mustAddUserToGroup(t *testing.T, userID, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.AddUserToGroup(userID, groupID, ""))
}

func (env *Env) mustMakeWebhook(t *testing.T, name string, channelID, creatorID uuid.UUID, secret string) model.Webhook {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	iconFileID, err := file.GenerateIconFile(env.FileManager, name)
	require.NoError(t, err)
	w, err := env.Repository.CreateWebhook(name, "", channelID, iconFileID, creatorID, secret)
	require.NoError(t, err)
	return w
}

func (env *Env) mustChangeChannelSubscription(t *testing.T, channelID, userID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.ChannelManager.ChangeChannelSubscriptions(channelID, map[uuid.UUID]model.ChannelSubscribeLevel{userID: model.ChannelSubscribeLevelMarkAndNotify}, false, uuid.Nil))
}
