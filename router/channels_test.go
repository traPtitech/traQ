package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
)

var (
	testUserID = "403807a5-cae6-453e-8a09-fc75d5b4ca91"
	engine     *xorm.Engine
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
	engine.DropTables("sessions", "channels", "users_private_channels")
	model.SetXORMEngine(engine)

	err = model.SyncSchema()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.Exit(code)
}

func beforeTest(t *testing.T) (*echo.Echo, *http.Cookie, echo.MiddlewareFunc) {
	engine.DropTables("sessions", "channels", "users_private_channels")
	err := model.SyncSchema()
	if err != nil {
		t.Fatalf("Failed to sync schema: %v", err)
	}
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(engine.DB().DB, "sessions", "/", 60*60*24*14, []byte("secret"))

	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := store.New(req, "sessions")
	sess.Values["userId"] = testUserID

	err = sess.Save(req, rec)
	if err != nil {
		t.Fatal(err)
	}
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
	mw := session.Middleware(store)

	return e, cookie, mw
}

func TestGetChannelsHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	for i := 0; i < 5; i++ {
		makeChannel(testUserID, "Channel-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannelsHandler), cookie, nil)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	var responseBody []ChannelForResponse
	err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatal("Failed to json parse ", err)
	}
}

func TestPostChannelsHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	postBody := PostChannel{
		ChannelType: "public",
		Name:        "test",
		Parent:      "",
	}

	body, err := json.Marshal(postBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostChannelsHandler), cookie, req)

	channelList, err := model.GetChannelList(testUserID)

	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusCreated {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	if len(channelList) != 1 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 1, len(channelList))
	}

	postBody = PostChannel{
		ChannelType: "private",
		Name:        "test",
		Parent:      "",
		Member: []string{
			testUserID,
			model.CreateUUID(),
		},
	}
	body, err = json.Marshal(postBody)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannelsHandler), cookie, req)
	channelList, err = model.GetChannelList(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(channelList) != 2 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 2, len(channelList))
	}

	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannelsHandler), cookie, req)
	channelList, err = model.GetChannelList(model.CreateUUID())
	if err != nil {
		t.Fatal(err)
	}

	if len(channelList) != 1 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 1, len(channelList))
	}
}

func TestGetChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(testUserID, "test", true)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)

	requestWithContext(t, mw(GetChannelsByChannelIDHandler), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	t.Log(rec.Body.String())
}

func TestPutChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(model.CreateUUID(), "test", true)

	req := httptest.NewRequest("PUT", "http://test", strings.NewReader(`{"name": "renamed"}`))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutChannelsByChannelIDHandler), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	channel, err := model.GetChannelByID(testUserID, channel.ID)
	if err != nil {
		t.Fatal(err)
	}

	if channel.Name != "renamed" {
		t.Fatalf("Channel name wrong: want %s, actual %s", "renamed", channel.Name)
	}

	if channel.UpdaterID != testUserID {
		t.Fatalf("Channel UpdaterId wrong: want %s, actual %s", testUserID, channel.UpdaterID)
	}

}

func TestDeleteChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(model.CreateUUID(), "test", true)

	req := httptest.NewRequest("DELETE", "http://test", strings.NewReader(`{"confirm": true}`))
	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelIDHandler), c)

	channel, err := model.GetChannelByID(testUserID, channel.ID)

	if err != nil {
		t.Fatal(err)
	}

	if !channel.IsDeleted {
		t.Fatal("Channel not deleted")
	}

	channelList, err := model.GetChannelList(testUserID)
	if len(channelList) != 0 {
		t.Fatal("Channel not deleted")
	}
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

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}

func makeChannel(userID, name string, isPublic bool) (*model.Channel, error) {
	channel := new(model.Channel)
	channel.CreatorID = userID
	channel.Name = name
	channel.IsPublic = isPublic
	err := channel.Create()
	return channel, err
}
