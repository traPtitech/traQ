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
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/role"
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

var envs = map[string]*Env{}

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
		env := &Env{}

		// テスト用データベース接続
		db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("%s%s", dbPrefix, key)))
		if err != nil {
			panic(err)
		}
		db.DB().SetMaxOpenConns(20)
		if err := migration.DropAll(db); err != nil {
			panic(err)
		}

		env.DB = db
		env.Hub = hub.New()
		env.SessStore = session.NewGormStore(db)

		// テスト用リポジトリ作成
		repo, err := repository.NewGormRepository(db, storage.NewInMemoryFileStorage(), env.Hub, zap.NewNop())
		if err != nil {
			panic(err)
		}
		if _, err := repo.Sync(); err != nil {
			panic(err)
		}
		env.Repository = repo

		// テスト用サーバー作成
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap(repo))

		r, err := rbac.New(db)
		if err != nil {
			panic(err)
		}
		handlers := &Handlers{
			RBAC:      r,
			Repo:      env.Repository,
			Hub:       env.Hub,
			SessStore: env.SessStore,
			Logger:    zap.NewNop(),
			Imaging: imaging.NewProcessor(imaging.Config{
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
		env.Server = httptest.NewServer(e)

		envs[key] = env
	}

	// テスト実行
	code := m.Run()

	// 後始末
	for _, env := range envs {
		env.Server.Close()
		env.DB.Close()
		env.Hub.Close()
	}
	os.Exit(code)
}

type Env struct {
	Server     *httptest.Server
	DB         *gorm.DB
	Repository repository.Repository
	Hub        *hub.Hub
	SessStore  session.Store
}

// Setup テストセットアップ
func Setup(t *testing.T, server string) *Env {
	t.Helper()
	env, ok := envs[server]
	if !ok {
		t.FailNow()
	}
	return env
}

// S 指定ユーザーのAPIセッショントークンを発行
func (env *Env) S(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	s, err := env.SessStore.IssueSession(userID, nil)
	require.NoError(t, err)
	return s.Token()
}

// R リクエストテスターを作成
func (env *Env) R(t *testing.T) *httpexpect.Expect {
	t.Helper()
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  env.Server.URL,
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
func (env *Env) CreateUser(t *testing.T, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Password: "testtesttesttest", Role: role.User})
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
