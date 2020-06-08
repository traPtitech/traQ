package oauth2

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	_ "github.com/go-sql-driver/mysql"
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
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const (
	dbPrefix = "traq-test-router-oauth2-"
	db1      = "db1"
	db2      = "db2"
	rand     = "random"
)

var envs = map[string]*Env{}

func TestMain(m *testing.M) {
	user := getEnvOrDefault("MARIADB_USERNAME", "root")
	pass := getEnvOrDefault("MARIADB_PASSWORD", "password")
	host := getEnvOrDefault("MARIADB_HOSTNAME", "127.0.0.1")
	port := getEnvOrDefault("MARIADB_PORT", "3306")
	dbs := []string{
		db1,
		db2,
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
		env.SessStore = session.NewMemorySessionStore()

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
		e.Use(extension.Wrap(repo, nil))

		r, err := rbac.New(db)
		if err != nil {
			panic(err)
		}
		config := &Handler{
			RBAC:      r,
			Repo:      env.Repository,
			SessStore: env.SessStore,
			Logger:    zap.NewNop(),
			Config: Config{
				AccessTokenExp:   1000,
				IsRefreshEnabled: true,
			},
		}
		config.Setup(e.Group("/oauth2"))
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

func (env *Env) IssueToken(t *testing.T, client *model.OAuth2Client, userID uuid.UUID, refresh bool) *model.OAuth2Token {
	t.Helper()
	token, err := env.Repository.IssueToken(client, userID, client.RedirectURI, client.Scopes, 1000, refresh)
	require.NoError(t, err)
	return token
}

func (env *Env) MakeAuthorizeData(t *testing.T, clientID string, userID uuid.UUID) *model.OAuth2Authorize {
	t.Helper()
	scopes := model.AccessScopes{}
	scopes.Add("read")
	authorize := &model.OAuth2Authorize{
		Code:           random.AlphaNumeric(36),
		ClientID:       clientID,
		UserID:         userID,
		CreatedAt:      time.Now(),
		ExpiresIn:      1000,
		RedirectURI:    "http://example.com",
		Scopes:         scopes,
		OriginalScopes: scopes,
		Nonce:          "nonce",
	}
	require.NoError(t, env.Repository.SaveAuthorize(authorize))
	return authorize
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}
