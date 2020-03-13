package v1

import (
	"bytes"
	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils"
	"go.uber.org/zap"
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
	"github.com/traPtitech/traQ/rbac/role"
)

const (
	random  = "random"
	common1 = "common1"
	common2 = "common2"
	common3 = "common3"
	common4 = "common4"
	common5 = "common5"
	common6 = "common6"
	common7 = "common7"
	s1      = "s1"
	s2      = "s2"
	s3      = "s3"
	s4      = "s4"
)

var (
	servers      = map[string]*httptest.Server{}
	repositories = map[string]*TestRepository{}
)

func TestMain(m *testing.M) {
	// setup server
	repos := []string{
		common1,
		common2,
		common3,
		common4,
		common5,
		common6,
		common7,
		s1,
		s2,
		s3,
		s4,
	}
	for _, key := range repos {
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.Binder = &extension.Binder{}
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap())

		repo := NewTestRepository()
		r, err := rbac.New(repo)
		if err != nil {
			panic(err)
		}
		h := hub.New()
		handlers := &Handlers{
			RBAC:             r,
			Repo:             repo,
			Hub:              h,
			Logger:           zap.NewNop(),
			Realtime:         realtime.NewService(h),
			AccessTokenExp:   1000,
			IsRefreshEnabled: true,
		}
		handlers.Setup(e.Group("/api"))
		servers[key] = httptest.NewServer(e)
		repositories[key] = repo
	}

	code := m.Run()

	for _, v := range servers {
		v.Close()
	}

	os.Exit(code)
}

func setup(t *testing.T, server string) (repository.Repository, *httptest.Server, *assert.Assertions, *require.Assertions, string, string) {
	t.Helper()
	s, ok := servers[server]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	repo := repositories[server]
	testUser := mustMakeUser(t, repo, random)
	adminUser, err := repo.GetUserByName("traq")
	require.NoError(err)
	return repo, s, assert, require, generateSession(t, testUser.ID), generateSession(t, adminUser.ID)
}

func setupWithUsers(t *testing.T, server string) (repository.Repository, *httptest.Server, *assert.Assertions, *require.Assertions, string, string, *model.User, *model.User) {
	t.Helper()
	s, ok := servers[server]
	if !ok {
		t.FailNow()
	}
	assert, require := assertAndRequire(t)
	repo := repositories[server]
	testUser := mustMakeUser(t, repo, random)
	adminUser, err := repo.GetUserByName("traq")
	require.NoError(err)
	return repo, s, assert, require, generateSession(t, testUser.ID), generateSession(t, adminUser.ID), testUser, adminUser
}

func assertAndRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func generateSession(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	require := require.New(t)
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()

	sess, err := sessions.Get(rec, req, true)
	require.NoError(err)
	require.NoError(sess.SetUser(userID))
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))[sessions.CookieName]

	return cookie.Value
}

func makeExp(t *testing.T, server *httptest.Server) *httpexpect.Expect {
	t.Helper()
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  server.URL,
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

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
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

func mustMakeUser(t *testing.T, repo repository.Repository, userName string) *model.User {
	t.Helper()
	if userName == random {
		userName = utils.RandAlphabetAndNumberString(32)
	}
	u, err := repo.CreateUser(userName, "test", role.User)
	require.NoError(t, err)
	return u
}

func mustMakeFile(t *testing.T, repo repository.Repository) *model.File {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := repo.SaveFile(repository.SaveFileArgs{
		FileName: "test.txt",
		FileSize: int64(buf.Len()),
		FileType: model.FileTypeUserFile,
		Src:      buf,
	})
	require.NoError(t, err)
	return f
}

func mustMakePin(t *testing.T, repo repository.Repository, messageID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	p, err := repo.CreatePin(messageID, userID)
	require.NoError(t, err)
	return p.ID
}

func mustMakeTag(t *testing.T, repo repository.Repository, userID uuid.UUID, tagText string) uuid.UUID {
	t.Helper()
	if tagText == random {
		tagText = utils.RandAlphabetAndNumberString(20)
	}
	tag, err := repo.GetOrCreateTagByName(tagText)
	require.NoError(t, err)
	require.NoError(t, repo.AddUserTag(userID, tag.ID))
	return tag.ID
}

func mustStarChannel(t *testing.T, repo repository.Repository, userID, channelID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddStar(userID, channelID))
}

func mustMakeUserGroup(t *testing.T, repo repository.Repository, name string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	g, err := repo.CreateUserGroup(name, "", "", adminID)
	require.NoError(t, err)
	return g
}

func mustAddUserToGroup(t *testing.T, repo repository.Repository, userID, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.AddUserToGroup(userID, groupID, ""))
}

func mustMakeWebhook(t *testing.T, repo repository.Repository, name string, channelID, creatorID uuid.UUID, secret string) model.Webhook {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	w, err := repo.CreateWebhook(name, "", channelID, creatorID, secret)
	require.NoError(t, err)
	return w
}

func mustMakeStamp(t *testing.T, repo repository.Repository, name string, userID uuid.UUID) *model.Stamp {
	t.Helper()
	if name == random {
		name = utils.RandAlphabetAndNumberString(20)
	}
	fileID, err := repo.GenerateIconFile(name)
	require.NoError(t, err)
	s, err := repo.CreateStamp(name, fileID, userID)
	require.NoError(t, err)
	return s
}

func mustIssueToken(t *testing.T, repo repository.Repository, client *model.OAuth2Client, userID uuid.UUID, refresh bool) *model.OAuth2Token {
	t.Helper()
	token, err := repo.IssueToken(client, userID, client.RedirectURI, client.Scopes, 1000, refresh)
	require.NoError(t, err)
	return token
}

func mustMakeAuthorizeData(t *testing.T, repo repository.Repository, clientID string, userID uuid.UUID) *model.OAuth2Authorize {
	t.Helper()
	scopes := model.AccessScopes{}
	scopes.Add("read")
	authorize := &model.OAuth2Authorize{
		Code:           utils.RandAlphabetAndNumberString(36),
		ClientID:       clientID,
		UserID:         userID,
		CreatedAt:      time.Now(),
		ExpiresIn:      1000,
		RedirectURI:    "http://example.com",
		Scopes:         scopes,
		OriginalScopes: scopes,
		Nonce:          "nonce",
	}
	require.NoError(t, repo.SaveAuthorize(authorize))
	return authorize
}

func mustChangeChannelSubscription(t *testing.T, repo repository.Repository, channelID, userID uuid.UUID) {
	t.Helper()
	require.NoError(t, repo.ChangeChannelSubscription(channelID, repository.ChangeChannelSubscriptionArgs{Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{userID: model.ChannelSubscribeLevelMarkAndNotify}}))
}

/*
func genPNG(salt string) []byte {
	if salt == random {
		salt = utils.RandAlphabetAndNumberString(20)
	}
	img := utils.GenerateIcon(salt)
	b := &bytes.Buffer{}
	_ = png.Encode(b, img)
	return b.Bytes()
}
*/
