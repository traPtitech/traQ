package v1

import (
	"bytes"
	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/utils/random"
	"go.uber.org/zap"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/jinzhu/gorm/dialects/mysql"
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
		env.Repository = NewTestRepository()
		env.Hub = hub.New()
		env.SessStore = session.NewMemorySessionStore()
		env.RBAC = newTestRBAC()

		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap(env.Repository))

		handlers := &Handlers{
			RBAC:      env.RBAC,
			Repo:      env.Repository,
			Hub:       env.Hub,
			Logger:    zap.NewNop(),
			OC:        counter.NewOnlineCounter(env.Hub),
			VM:        viewer.NewManager(env.Hub),
			SessStore: env.SessStore,
			Imaging: imaging.NewProcessor(imaging.Config{
				MaxPixels:        1000 * 1000,
				Concurrency:      1,
				ThumbnailMaxSize: image.Pt(360, 480),
				ImageMagickPath:  "",
			}),
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
	Server     *httptest.Server
	Repository repository.Repository
	Hub        *hub.Hub
	SessStore  session.Store
	RBAC       rbac.RBAC
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
			Timeout: time.Second * 30,
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
	ch, err := env.Repository.CreatePublicChannel(name, uuid.Nil, uuid.Nil)
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
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Password: "test", Role: role.User})
	require.NoError(t, err)
	return u
}

func (env *Env) mustMakeFile(t *testing.T) model.FileMeta {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := env.Repository.SaveFile(repository.SaveFileArgs{
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
	g, err := env.Repository.CreateUserGroup(name, "", "", adminID)
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
	w, err := env.Repository.CreateWebhook(name, "", channelID, creatorID, secret)
	require.NoError(t, err)
	return w
}

func (env *Env) mustMakeStamp(t *testing.T, name string, userID uuid.UUID) *model.Stamp {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	fileID, err := repository.GenerateIconFile(env.Repository, name)
	require.NoError(t, err)
	s, err := env.Repository.CreateStamp(repository.CreateStampArgs{Name: name, FileID: fileID, CreatorID: userID})
	require.NoError(t, err)
	return s
}

func (env *Env) mustChangeChannelSubscription(t *testing.T, channelID, userID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.ChangeChannelSubscription(channelID, repository.ChangeChannelSubscriptionArgs{Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{userID: model.ChannelSubscribeLevelMarkAndNotify}}))
}

type rbacImpl struct {
	roles role.Roles
}

func newTestRBAC() rbac.RBAC {
	rbac := &rbacImpl{
		roles: role.GetSystemRoles(),
	}
	return rbac
}

func (rbacImpl *rbacImpl) IsGranted(r string, p permission.Permission) bool {
	if r == role.Admin {
		return true
	}
	return rbacImpl.roles.HasAndIsGranted(r, p)
}

func (rbacImpl *rbacImpl) IsAllGranted(roles []string, perm permission.Permission) bool {
	for _, role := range roles {
		if !rbacImpl.IsGranted(role, perm) {
			return false
		}
	}
	return true
}

func (rbacImpl *rbacImpl) IsAnyGranted(roles []string, perm permission.Permission) bool {
	for _, role := range roles {
		if rbacImpl.IsGranted(role, perm) {
			return true
		}
	}
	return false
}

func (rbacImpl *rbacImpl) GetGrantedPermissions(roleName string) []permission.Permission {
	ro, ok := rbacImpl.roles[roleName]
	if ok {
		return ro.Permissions().Array()
	}
	return nil
}
