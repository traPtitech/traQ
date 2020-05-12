package v3

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/sessions"
	imaging2 "github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	dbPrefix = "traq-test-router-v3-"
	common   = "common"
	rand     = "random"
)

var (
	servers      = map[string]*httptest.Server{}
	dbConns      = map[string]*gorm.DB{}
	repositories = map[string]repository.Repository{}
	hubs         = map[string]*hub.Hub{}
)

func TestMain(m *testing.M) {
	user := getEnvOrDefault("MARIADB_USERNAME", "root")
	pass := getEnvOrDefault("MARIADB_PASSWORD", "password")
	host := getEnvOrDefault("MARIADB_HOSTNAME", "127.0.0.1")
	port := getEnvOrDefault("MARIADB_PORT", "3306")
	dbs := []string{
		common,
	}
	if err := migration.CreateDatabasesIfNotExists("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true", user, pass, host, port), dbPrefix, dbs...); err != nil {
		panic(err)
	}

	for _, key := range dbs {
		// テスト用データベース接続
		db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("%s%s", dbPrefix, key)))
		if err != nil {
			panic(err)
		}
		db.DB().SetMaxOpenConns(20)
		if err := migration.DropAll(db); err != nil {
			panic(err)
		}
		dbConns[key] = db

		hub := hub.New()
		hubs[key] = hub

		// テスト用リポジトリ作成
		repo, err := repository.NewGormRepository(db, storage.NewInMemoryFileStorage(), hub, zap.NewNop())
		if err != nil {
			panic(err)
		}
		if _, err := repo.Sync(); err != nil {
			panic(err)
		}
		repositories[key] = repo

		// テスト用サーバー作成
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap(repo))

		r, err := rbac.New(repo)
		if err != nil {
			panic(err)
		}
		handlers := &Handlers{
			RBAC:   r,
			Repo:   repo,
			WS:     nil,
			Hub:    hub,
			Logger: zap.NewNop(),
			Imaging: imaging2.NewProcessor(imaging2.Config{
				MaxPixels:        1000 * 1000,
				Concurrency:      1,
				ThumbnailMaxSize: image.Pt(360, 480),
				ImageMagickPath:  "",
			}),
			Config: Config{
				Version:  "version",
				Revision: "revision",
			},
		}
		handlers.Setup(e.Group("/api"))
		servers[key] = httptest.NewServer(e)
	}

	// テスト実行
	code := m.Run()

	// 後始末
	for _, v := range servers {
		v.Close()
	}
	for _, v := range dbConns {
		v.Close()
	}
	for _, v := range hubs {
		v.Close()
	}
	os.Exit(code)
}

// Setup テストセットアップ
func Setup(t *testing.T, server string) (repository.Repository, *httptest.Server) {
	t.Helper()
	s, ok := servers[server]
	if !ok {
		t.FailNow()
	}
	repo := repositories[server]
	return repo, s
}

// S 指定ユーザーのAPIセッショントークンを発行
func S(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	require := require.New(t)

	sess, err := sessions.IssueNewSession("127.0.0.1", "test")
	require.NoError(err)
	require.NoError(sess.SetUser(userID))
	return sess.GetToken()
}

// R リクエストテスターを作成
func R(t *testing.T, server *httptest.Server) *httpexpect.Expect {
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

// CreateUser ユーザーを必ず作成します
func CreateUser(t *testing.T, repo repository.Repository, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	u, err := repo.CreateUser(repository.CreateUserArgs{Name: userName, Password: "testtesttesttest", Role: role.User})
	require.NoError(t, err)
	return u
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}
