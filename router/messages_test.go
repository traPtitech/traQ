package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
	"github.com/traPtitech/traQ/model"
)

var (
	testUserId    = ""
	testChannelId = ""
	sampleText    = "popopo"
)

func TestMain(m *testing.M) {
	os.Setenv("MARIADB_DATABASE", "traq-test-router")
	code := m.Run()
	os.Exit(code)
}

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
	sess, err := store.New(req, "sessions")

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

func TestGetMessagesByChannelIdHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	defer model.Close()

	for i := 0; i < 5; i++ {
		makeMessage()
	}

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/channels/:channelId/messages")
	c.SetParamNames("channelId")
	c.SetParamValues(testChannelId)
	requestWithContext(t, mw(GetMessagesByChannelIdHandler), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	var responseBody []MessageForResponse
	err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatal(err)
	}

	if len(responseBody) != 5 {
		t.Errorf("No found all messages: want %d, actual %d", 5, len(responseBody))
	}

}

func TestPostMessageHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	defer model.Close()

	post := requestMessage{
		Text: "test message",
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostMessageHandler), cookie, req)

	message := &MessageForResponse{}

	result, err := ioutil.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(result, message)
	if err != nil {
		t.Fatal(err)
	}

	if message.Content != post.Text {
		t.Fatal("message text is wrong: want %v, actual %v", post.Text, message.Content)
	}

	if rec.Code != http.StatusCreated {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}
}

func TestPutMessageByIdHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	defer model.Close()

	message := makeMessage()

	post := requestMessage{
		Text: "test message",
	}
	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageId")
	c.SetParamNames("messageId")
	c.SetParamValues(message.Id)
	requestWithContext(t, mw(PutMessageByIdHandler), c)

	message, err = model.GetMessage(message.Id)
	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	if message.Text != post.Text {
		t.Fatalf("message text is wrong: want %v, actual %v", post.Text, message.Text)
	}

}

func TestDeleteMessageByIdHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	defer model.Close()

	message := makeMessage()

	req := httptest.NewRequest("DELETE", "http://test", nil)

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageId")
	c.SetParamNames("messageId")
	c.SetParamValues(message.Id)
	requestWithContext(t, mw(DeleteMessageByIdHandler), c)

	message, err := model.GetMessage(message.Id)
	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	if message.IsDeleted != true {
		t.Fatalf("message text is wrong: want %v, actual %v", true, message.IsDeleted)
	}

}

func makeMessage() *model.Messages {
	message := &model.Messages{
		UserId:    testUserId,
		ChannelId: testChannelId,
		Text:      "popopo",
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

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}
