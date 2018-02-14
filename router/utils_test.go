package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
)

var (
	testUser = &model.User{
		Name:  "testUser",
		Email: "example@trap.jp",
		Icon:  "empty",
	}
	testChannelID = ""
	engine        *xorm.Engine
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

	dbname := "traq-test-router"

	var err error
	engine, err = xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=true", user, pass, host, dbname))
	if err != nil {
		panic(err)
	}
	defer engine.Close()

	engine.ShowSQL(false)
	engine.DropTables("sessions", "messages", "users_private_channels", "channels", "users", "clips", "stars", "tags", "users_tags")
	engine.SetMapper(core.GonicMapper{})
	model.SetXORMEngine(engine)

	err = model.SyncSchema()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.Exit(code)
}

func beforeTest(t *testing.T) (*echo.Echo, *http.Cookie, echo.MiddlewareFunc) {
	engine.DropTables("sessions", "messages", "users_private_channels", "channels", "users", "clips", "stars", "tags", "users_tags")
	if err := model.SyncSchema(); err != nil {
		t.Fatalf("Failed to sync schema: %v", err)
	}
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(engine.DB().DB, "sessions", "/", 60*60*24*14, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	if err := testUser.SetPassword("test"); err != nil {
		t.Fatalf("an error occurred while setting password: %v", err)
	}
	if err := testUser.Create(); err != nil {
		t.Fatalf("an error occurred while creating user: %v", err)
	}
	testChannelID = model.CreateUUID()

	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := store.New(req, "sessions")

	sess.Values["userID"] = testUser.ID
	if err := sess.Save(req, rec); err != nil {
		t.Fatal(err)
	}
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
	mw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return session.Middleware(store)(GetUserInfo(next))(c)
		}

	}
	return e, cookie, mw
}

func beforeLoginTest(t *testing.T) (*echo.Echo, echo.MiddlewareFunc) {
	engine.DropTables("sessions", "messages", "users_private_channels", "channels", "users", "clips", "stars")
	if err := model.SyncSchema(); err != nil {
		t.Fatalf("Failed to sync schema: %v", err)
	}
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(engine.DB().DB, "sessions", "/", 60*60*24*14, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(echo.GET, "/", nil)
	if _, err := store.New(req, "sessions"); err != nil {
		t.Fatal(err)
	}

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
		t.Fatal(err)
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

func makeChannel(userID, name string, isPublic bool) (*model.Channel, error) {
	channel := new(model.Channel)
	channel.CreatorID = userID
	channel.Name = name
	channel.IsPublic = isPublic
	err := channel.Create()
	return channel, err
}

func makeMessage() *model.Message {
	message := &model.Message{
		UserID:    testUser.ID,
		ChannelID: testChannelID,
		Text:      "popopo",
	}
	message.Create()
	return message
}

func makeTag(userID, tagText string) (*model.UsersTag, error) {
	tag := &model.UsersTag{
		UserID: userID,
	}
	err := tag.Create(tagText)
	return tag, err
}

func clipMessage(userID, messageID string) error {
	clip := &model.Clip{
		UserID:    userID,
		MessageID: messageID,
	}

	return clip.Create()
}

<<<<<<< HEAD
func starChannel(userID, channelID string) error {
	star := &model.Star{
		UserID:    userID,
		ChannelID: channelID,
	}

	return star.Create()
=======
func pinMessage(channelID, messageID, userID string) *model.Pins {
	pin := &model.Pin{
		ChannelID: channelID,
		MessageID: messageID,
		UserID:    userID,
	}

	return pin.Create()

>>>>>>> update
}

func createUser(t *testing.T) {
	user := &model.User{
		Name:  "PostLogin",
		Email: "example@trap.jp",
		Icon:  "empty",
	}
	err := user.SetPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	err = user.Create()
	if err != nil {
		t.Fatal(err)
	}
}
