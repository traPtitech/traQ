package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
)

var (
	testUserID = "403807a5-cae6-453e-8a09-fc75d5b4ca91"
)

func beforeTest(t *testing.T) (*echo.Echo, *http.Cookie, echo.MiddlewareFunc) {
	model.BeforeTest(t)
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(model.GetSQLDB(), "sessions", "/", 60*60*24*14, []byte("secret"))

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
	defer model.Close()

	for i := 0; i < 5; i++ {
		makeChannel(testUserID, "Channel-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannelsHandler), cookie)

	var responseBody []ChannelForResponse
	err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatal("Failed to json parse ", err)
	}
	fmt.Println(responseBody)
}

func TestPostChannelsHandler(test *testing.T) {
}

func TestGetChannelsByChannelIdHandler(test *testing.T) {
}

func TestPutChannelsByChannelIdHandler(test *testing.T) {
}

func TestDeleteChannelsByChannelIdHandler(test *testing.T) {
}

func request(e *echo.Echo, t *testing.T, handler echo.HandlerFunc, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "http://test", nil)

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

func makeChannel(userId, name string, isPublic bool) error {
	channel := new(model.Channels)
	channel.CreatorId = userId
	channel.Name = name
	channel.IsPublic = isPublic
	return channel.Create()
}
