package router

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/traPtitech/traQ/external/storage"
	"github.com/traPtitech/traQ/utils/validator"

	"github.com/stretchr/testify/assert"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
)

var (
	testUser *model.User
	db       *gorm.DB
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
	db = engine
	model.SetGORMEngine(engine)

	// テストで作成されたfileは全てメモリ上に乗ります。容量注意
	model.SetFileManager("", storage.NewInMemoryFileManager())

	if err := model.Sync(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func beforeTest(t *testing.T) (*echo.Echo, *http.Cookie, echo.MiddlewareFunc, *assert.Assertions, *require.Assertions) {
	require := require.New(t)

	require.NoError(model.DropTables())
	require.NoError(model.Sync())
	e := echo.New()
	e.Validator = validator.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(db.DB(), "sessions", "/", 60*60*24*14, []byte("secret"))
	require.NoError(err)

	testUser = mustCreateUser(t, "testUser")

	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := store.New(req, "sessions")
	require.NoError(err)

	sess.Values["userID"] = testUser.GetUID()
	require.NoError(sess.Save(req, rec))

	cookie := parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
	mw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return session.Middleware(store)(UserAuthenticate(nil)(next))(c)
		}

	}
	return e, cookie, mw, assert.New(t), require
}

func beforeLoginTest(t *testing.T) (*echo.Echo, echo.MiddlewareFunc) {
	require := require.New(t)

	require.NoError(model.DropTables())
	require.NoError(model.Sync())
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(db.DB(), "sessions", "/", 60*60*24*14, []byte("secret"))
	require.NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)
	_, err = store.New(req, "sessions")
	require.NoError(err)

	mw := session.Middleware(store)
	return e, mw
}

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}

func requestWithContext(t *testing.T, handler echo.HandlerFunc, c echo.Context) {
	err := handler(c)

	if err != nil {
		t.Fatal(err)
	}
}

func request(e *echo.Echo, t *testing.T, handler echo.HandlerFunc, cookie *http.Cookie, req *http.Request) *httptest.ResponseRecorder {
	if req == nil {
		req = httptest.NewRequest("GET", "http://test", nil)
	}

	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if cookie != nil {
		req.Header.Add("Cookie", fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	if err != nil {
		rec.Code = err.(*echo.HTTPError).Code
	}

	return rec
}

func getContext(e *echo.Echo, t *testing.T, cookie *http.Cookie, req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	if req == nil {
		req = httptest.NewRequest("GET", "http://test", nil)
	}

	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if cookie != nil {
		req.Header.Add("Cookie", fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func mustMakeChannelDetail(t *testing.T, userID uuid.UUID, name, parentID string, isPublic bool) *model.Channel {
	ch, err := model.CreateChannel(parentID, name, userID, isPublic)
	require.NoError(t, err)
	return ch
}

func mustMakePrivateChannel(t *testing.T, userID1, userID2 uuid.UUID, name string) *model.Channel {
	channel := mustMakeChannelDetail(t, userID1, name, "", false)
	require.NoError(t, model.AddPrivateChannelMember(channel.GetCID(), userID1))
	if userID1 != userID2 {
		require.NoError(t, model.AddPrivateChannelMember(channel.GetCID(), userID2))
	}
	return channel
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
