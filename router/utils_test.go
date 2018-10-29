package router

import (
	"bytes"
	"fmt"
	"github.com/gavv/httpexpect"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/external/storage"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
)

var (
	testUser *model.User
	server   *httptest.Server
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

	dbname := "traq-test-router"
	config.DatabaseName = "traq-test-router"

	engine, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, dbname))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	model.SetGORMEngine(engine)

	// テストで作成されたfileは全てメモリ上に乗ります。容量注意
	model.SetFileManager("", storage.NewInMemoryFileManager())

	// setup server
	e := echo.New()
	SetupRouting(e, &Handlers{})
	server = httptest.NewServer(e)
	defer server.Close()

	os.Exit(m.Run())
}

func beforeTest(t *testing.T) (*assert.Assertions, *require.Assertions, string, string) {
	require := require.New(t)
	assert := assert.New(t)

	require.NoError(model.DropTables())
	_, err := model.Sync()
	require.NoError(err)

	testUser = mustCreateUser(t, "testUser")

	return assert, require, generateSession(t, testUser.GetUID()), generateSession(t, model.ServerUser().GetUID())
}

func generateSession(t *testing.T, userID uuid.UUID) string {
	require := require.New(t)
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()

	sess, err := sessions.Get(rec, req, true)
	require.NoError(err)
	require.NoError(sess.SetUser(userID))
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))[sessions.CookieName]

	return cookie.Value
}

func makeExp(t *testing.T) *httpexpect.Expect {
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

func mustMakeChannelDetail(t *testing.T, userID uuid.UUID, name, parentID string) *model.Channel {
	ch, err := model.CreatePublicChannel(parentID, name, userID)
	require.NoError(t, err)
	return ch
}

func mustMakePrivateChannel(t *testing.T, name string, members []uuid.UUID) *model.Channel {
	ch, err := model.CreatePrivateChannel("", name, members[0], members)
	require.NoError(t, err)
	return ch
}

func mustMakeMessage(t *testing.T, userID, channelID uuid.UUID) *model.Message {
	m, err := model.CreateMessage(userID, channelID, "popopo")
	require.NoError(t, err)
	return m
}

func mustMakeTag(t *testing.T, userID uuid.UUID, tagText string) uuid.UUID {
	tag, err := model.GetOrCreateTagByName(tagText)
	require.NoError(t, err)
	require.NoError(t, model.AddUserTag(userID, tag.GetID()))
	return tag.GetID()
}

func mustMakeUnread(t *testing.T, userID, messageID uuid.UUID) {
	require.NoError(t, model.SetMessageUnread(userID, messageID))
}

func mustStarChannel(t *testing.T, userID, channelID uuid.UUID) {
	require.NoError(t, model.AddStar(userID, channelID))
}

func mustMakePin(t *testing.T, userID, messageID uuid.UUID) uuid.UUID {
	id, err := model.CreatePin(messageID, userID)
	require.NoError(t, err)
	return id
}

func mustCreateUser(t *testing.T, name string) *model.User {
	u, err := model.CreateUser(name, name+"@test.test", "test", role.User)
	require.NoError(t, err)
	return u
}

func mustMakeFile(t *testing.T) *model.File {
	file := &model.File{
		Name:      "test.txt",
		Size:      90,
		CreatorID: testUser.ID,
	}
	require.NoError(t, file.Create(bytes.NewBufferString("test message")))
	return file
}
