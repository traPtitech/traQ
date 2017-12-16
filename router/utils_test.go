package router

import(
	"os"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/go-xorm/core"
	"github.com/labstack/echo-contrib/session"
	"github.com/srinathgs/mysqlstore"
)

var (
	engine *xorm.Engine
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
	engine.DropTables("sessions", "messages")
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
	testChannelID = model.CreateUUID()
	testUserID = model.CreateUUID()

	engine.DropTables("sessions","messages")
	if err := model.SyncSchema(); err != nil {
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
	if err := sess.Save(req, rec); err != nil {
		t.Fatal(err)
	}
	cookie := parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
	mw := session.Middleware(store)

	return e, cookie, mw
}