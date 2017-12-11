package router

import (
	"fmt"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
)

var (
	testUserId = ""
	testChannelId = ""
	sampleText = "popopo"
)



func beforeTest(t *testing.T) (*echo.Echo, *http.Cookie, echo.MiddlewareFunc) {
	testChannelId = model.CreateUUID()
	testUserId = model.CreateUUID()

	model.BeforeTest(t)
	e := echo.New()

	store, err := mysqlstore.NewMySQLStoreFromConnection(model.GetSQLDB(), "sessions", "/", 60*60*24*14, []byte("secret"))

	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := store.New(req,"sessions")

	sess.Values["userId"] = testUserId
	if err := sess.Save(req, rec); err != nil {
		t.Fatal(err)
	}
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
	mw := session.Middleware(store)

	return e, cookie, mw
}

func TestGetMessageByIdHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	defer model.Close()

	message := makeMessage()

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/messages/:messageId")
	c.SetParamNames("messageId")
	c.SetParamValues(message.Id)

	requestWithContext(t, mw(GetMessageByIdHandler), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}
	t.Log(rec.Body.String())
}

func TestMessagesByChannelIdHandler(t *testing.T) {	
}

func TestPostMessageHandler(t *testing.T) {

}

func TestPutMessageByIdHandler(t *testing.T) {

}

func TestDeleteMessageByIdHandler(t *testing.T) {

}

func makeMessage() *model.Messages {
	message := &model.Messages{
		UserId : testUserId,
		ChannelId : testChannelId,
		Text : "popopo",
	}
	message.Create()
	return message
}

func requestWithContext(t *testing.T, handler echo.HandlerFunc, c echo.Context) {
	err := handler(c)

	if err != nil {
		t.Fatal(err)
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
	c := e.NewContext(req,rec)
	return c, rec

}


func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}