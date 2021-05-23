package v3

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"
)

const (
	dbPrefix = "traq-test-router-v3-"
	common1  = "common1"
	s1       = "s1"
	rand     = "random"
)

var envs = map[string]*Env{}

func TestMain(m *testing.M) {
	user := getEnvOrDefault("MARIADB_USERNAME", "root")
	pass := getEnvOrDefault("MARIADB_PASSWORD", "password")
	host := getEnvOrDefault("MARIADB_HOSTNAME", "127.0.0.1")
	port := getEnvOrDefault("MARIADB_PORT", "3306")
	dbs := []string{
		common1,
		s1,
	}
	if err := migration.CreateDatabasesIfNotExists("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true", user, pass, host, port), dbPrefix, dbs...); err != nil {
		panic(err)
	}

	for _, key := range dbs {
		env := &Env{}

		// テスト用データベース接続
		engine, err := gorm.Open(mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true", user, pass, host, port, fmt.Sprintf("%s%s", dbPrefix, key)),
		}))
		if err != nil {
			panic(err)
		}
		db, err := engine.DB()
		if err != nil {
			panic(err)
		}
		db.SetMaxOpenConns(20)
		engine.Logger = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
		})
		if err := migration.DropAll(engine); err != nil {
			panic(err)
		}

		env.DB = engine
		env.Hub = hub.New()
		env.SessStore = session.NewMemorySessionStore()

		// テスト用リポジトリ作成
		repo, err := repository.NewGormRepository(engine, env.Hub, zap.NewNop())
		if err != nil {
			panic(err)
		}
		if init, err := repo.Sync(); err != nil {
			panic(err)
		} else if init {
			// システムユーザーロール投入
			if err := repo.CreateUserRoles(role.SystemRoleModels()...); err != nil {
				panic(err)
			}
		}
		env.Repository = repo

		env.CM, _ = channel.InitChannelManager(repo, zap.NewNop())
		env.MM, _ = message.NewMessageManager(repo, env.CM, zap.NewNop())
		env.IP = imaging.NewProcessor(imaging.Config{
			MaxPixels:        1000 * 1000,
			Concurrency:      1,
			ThumbnailMaxSize: image.Pt(360, 480),
			ImageMagickPath:  "",
		})
		env.FM, _ = file.InitFileManager(repo, storage.NewInMemoryFileStorage(), env.IP, zap.NewNop())

		// テスト用サーバー作成
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(zap.NewNop())
		e.Use(extension.Wrap(repo, env.CM))

		r, err := rbac.New(repo)
		if err != nil {
			panic(err)
		}
		handlers := &Handlers{
			RBAC:           r,
			Repo:           env.Repository,
			Hub:            env.Hub,
			SessStore:      env.SessStore,
			ChannelManager: env.CM,
			MessageManager: env.MM,
			FileManager:    env.FM,
			Logger:         zap.NewNop(),
			Imaging:        env.IP,
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
		db, _ := env.DB.DB()
		_ = db.Close()
		env.Hub.Close()
	}
	os.Exit(code)
}

type Env struct {
	Server     *httptest.Server
	DB         *gorm.DB
	Repository repository.Repository
	CM         channel.Manager
	MM         message.Manager
	FM         file.Manager
	IP         imaging.Processor
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
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Password: "testtesttesttest", Role: role.User, IconFileID: uuid.Must(uuid.NewV4())})
	require.NoError(t, err)
	return u
}

// CreateChannel チャンネルを必ず作成します
func (env *Env) CreateChannel(t *testing.T, name string) *model.Channel {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	ch, err := env.CM.CreatePublicChannel(name, uuid.Nil, uuid.Nil)
	require.NoError(t, err)
	return ch
}

// CreateMessage メッセージを必ず作成します
func (env *Env) CreateMessage(t *testing.T, userID, channelID uuid.UUID, text string) *model.Message {
	t.Helper()
	if text == rand {
		text = random.AlphaNumeric(20)
	}
	m, err := env.Repository.CreateMessage(userID, channelID, text)
	require.NoError(t, err)
	return m
}

// MakeFile ファイルを必ず作成します
func (env *Env) MakeFile(t *testing.T) model.File {
	t.Helper()
	buf := bytes.NewBufferString("test message")
	f, err := env.FM.Save(file.SaveArgs{
		FileName: "test.txt",
		FileSize: int64(buf.Len()),
		FileType: model.FileTypeUserFile,
		Src:      buf,
	})
	require.NoError(t, err)
	return f
}

func (env *Env) CreateBot(t *testing.T, name string, creatorID uuid.UUID) *model.Bot {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	m, err := env.Repository.CreateBot(name, "po", "totall a desc", uuid.Nil, creatorID, "https://example.com")
	require.NoError(t, err)
	return m
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}
