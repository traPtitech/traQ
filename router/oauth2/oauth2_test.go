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
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/sessions"
	random2 "github.com/traPtitech/traQ/utils/random"
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
	random   = "random"
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
		db1,
		db2,
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
		config := &Config{
			RBAC:             r,
			Repo:             repo,
			Logger:           zap.NewNop(),
			AccessTokenExp:   1000,
			IsRefreshEnabled: true,
		}
		config.Setup(e.Group("/oauth2"))
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
	if userName == random {
		userName = random2.AlphaNumeric(32)
	}
	u, err := repo.CreateUser(repository.CreateUserArgs{Name: userName, Password: "testtesttesttest", Role: role.User})
	require.NoError(t, err)
	return u
}

func IssueToken(t *testing.T, repo repository.Repository, client *model.OAuth2Client, userID uuid.UUID, refresh bool) *model.OAuth2Token {
	t.Helper()
	token, err := repo.IssueToken(client, userID, client.RedirectURI, client.Scopes, 1000, refresh)
	require.NoError(t, err)
	return token
}

func MakeAuthorizeData(t *testing.T, repo repository.Repository, clientID string, userID uuid.UUID) *model.OAuth2Authorize {
	t.Helper()
	scopes := model.AccessScopes{}
	scopes.Add("read")
	authorize := &model.OAuth2Authorize{
		Code:           random2.AlphaNumeric(36),
		ClientID:       clientID,
		UserID:         userID,
		CreatedAt:      time.Now(),
		ExpiresIn:      1000,
		RedirectURI:    "http://example.com",
		Scopes:         scopes,
		OriginalScopes: scopes,
		Nonce:          "nonce",
	}
	require.NoError(t, repo.SaveAuthorize(authorize))
	return authorize
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}
