package v3

import (
	"bytes"
	"fmt"
	"image"
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

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	gorm2 "github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/storage"
)

const (
	dbPrefix = "traq-test-router-v3-"
	common1  = "common1"
	s1       = "s1"
	s2       = "s2"
	s3       = "s3"
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
		s2,
		s3,
	}
	if err := migration.CreateDatabasesIfNotExists("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true", user, pass, host, port), dbPrefix, dbs...); err != nil {
		panic(err)
	}

	for _, key := range dbs {
		env := &Env{}

		l := zap.NewNop()

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
		engine.Logger = gormzap.New(l)
		if err := migration.DropAll(engine); err != nil {
			panic(err)
		}

		env.DB = engine
		env.Hub = hub.New()
		env.SessStore = session.NewMemorySessionStore()

		// テスト用リポジトリ作成
		repo, init, err := gorm2.NewGormRepository(engine, env.Hub, l.Named("repository"), true)
		if err != nil {
			panic(err)
		}
		if init {
			// システムユーザーロール投入
			if err := repo.CreateUserRoles(role.SystemRoleModels()...); err != nil {
				panic(err)
			}
		}
		env.Repository = repo

		env.CM, _ = channel.InitChannelManager(repo, l.Named("CM"))
		env.MM, _ = message.NewMessageManager(repo, env.CM, l.Named("MM"))
		env.IP = imaging.NewProcessor(imaging.Config{
			MaxPixels:        1000 * 1000,
			Concurrency:      1,
			ThumbnailMaxSize: image.Pt(360, 480),
		})
		env.FM, _ = file.InitFileManager(repo, storage.NewInMemoryFileStorage(), env.IP, l.Named("FM"))

		// テスト用サーバー作成
		e := echo.New()
		e.HideBanner = true
		e.HidePort = true
		e.HTTPErrorHandler = extension.ErrorHandler(l)
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
			Logger:         l,
			Imaging:        env.IP,
			Config: Config{
				Version:         "version",
				Revision:        "revision",
				SkyWaySecretKey: "dummy.secret.key",
				AllowSignUp:     false,
				EnabledExternalAccountProviders: map[string]bool{
					"traq": true,
				},
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
	SE         search.Engine
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
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Password: "!test_test@test-", Role: role.User, IconFileID: uuid.Must(uuid.NewV4())})
	require.NoError(t, err)
	return u
}

// CreateAdmin Adminユーザーを必ず作成します
func (env *Env) CreateAdmin(t *testing.T, userName string) model.UserInfo {
	t.Helper()
	if userName == rand {
		userName = random.AlphaNumeric(32)
	}
	u, err := env.Repository.CreateUser(repository.CreateUserArgs{Name: userName, Password: "!test_test@test-", Role: role.Admin, IconFileID: uuid.Must(uuid.NewV4())})
	require.NoError(t, err)
	return u
}

// AddTag ユーザーに必ずタグを追加します
func (env *Env) AddTag(t *testing.T, name string, userID uuid.UUID) model.UserTag {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}

	tag, err := env.Repository.GetOrCreateTag(name)
	require.NoError(t, err)

	require.NoError(t, env.Repository.AddUserTag(userID, tag.ID))

	ut, err := env.Repository.GetUserTag(userID, tag.ID)
	require.NoError(t, err)
	return ut
}

// CreateUserGroup ユーザーグループを必ず作成します
func (env *Env) CreateUserGroup(t *testing.T, name, description, groupType string, adminID uuid.UUID) *model.UserGroup {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	icon := env.CreateFile(t, uuid.Nil, uuid.Nil)
	ug, err := env.Repository.CreateUserGroup(name, description, groupType, adminID, icon.GetID())
	require.NoError(t, err)
	return ug
}

// AddUserToUserGroup ユーザーをユーザーグループに必ず追加します
func (env *Env) AddUserToUserGroup(t *testing.T, userID, groupID uuid.UUID, role string) {
	t.Helper()
	require.NoError(t, env.Repository.AddUserToGroup(userID, groupID, role))
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

// AddStar 指定したチャンネルをスターします
func (env *Env) AddStar(t *testing.T, userID, channelID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.AddStar(userID, channelID))
}

// CreateDMChannel DMチャンネルを必ず作成します
func (env *Env) CreateDMChannel(t *testing.T, user1, user2 uuid.UUID) *model.Channel {
	dm, err := env.CM.GetDMChannel(user1, user2)
	require.NoError(t, err)
	return dm
}

// CreateMessage メッセージを必ず作成します
func (env *Env) CreateMessage(t *testing.T, userID, channelID uuid.UUID, text string) message.Message {
	t.Helper()
	if text == rand {
		text = random.AlphaNumeric(20)
	}
	m, err := env.MM.Create(channelID, userID, text)
	require.NoError(t, err)
	return m
}

// MakeMessageUnread 指定したメッセージを未読にします
func (env *Env) MakeMessageUnread(t *testing.T, userID, messageID uuid.UUID) {
	t.Helper()
	require.NoError(t, env.Repository.SetMessageUnread(userID, messageID, false))
}

// CreateStamp スタンプを必ず作成します
func (env *Env) CreateStamp(t *testing.T, creator uuid.UUID, name string) *model.Stamp {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	f := env.CreateFile(t, creator, uuid.Nil)
	s, err := env.Repository.CreateStamp(repository.CreateStampArgs{
		Name:      name,
		FileID:    f.GetID(),
		CreatorID: creator,
	})
	require.NoError(t, err)
	return s
}

// CreateStampPalette スタンプパレットを必ず作成します
func (env *Env) CreateStampPalette(t *testing.T, creator uuid.UUID, name string, stamps model.UUIDs) *model.StampPalette {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	sp, err := env.Repository.CreateStampPalette(name, "desc", stamps, creator)
	require.NoError(t, err)
	return sp
}

// AddStampToMessage メッセージにスタンプを必ず押します
func (env *Env) AddStampToMessage(t *testing.T, messageID, stampID, userID uuid.UUID) {
	t.Helper()
	_, err := env.MM.AddStamps(messageID, stampID, userID, 1)
	require.NoError(t, err)
}

// CreateFile ファイルを必ず作成します
func (env *Env) CreateFile(t *testing.T, creatorID, channelID uuid.UUID) model.File {
	return env.CreateFileWithName(t, creatorID, channelID, "test.txt")
}

// CreateFileWithName ファイルを必ず作成します
func (env *Env) CreateFileWithName(t *testing.T, creatorID, channelID uuid.UUID, filename string) model.File {
	t.Helper()

	var cr, ch optional.Of[uuid.UUID]
	if creatorID != uuid.Nil {
		cr = optional.From(creatorID)
	}
	if channelID != uuid.Nil {
		ch = optional.From(channelID)
	}

	buf := bytes.NewBufferString("test message")
	args := file.SaveArgs{
		FileName:  filename,
		FileSize:  int64(buf.Len()),
		FileType:  model.FileTypeUserFile,
		CreatorID: cr,
		ChannelID: ch,
		Src:       buf,
	}

	members, err := env.CM.GetDMChannelMembers(channelID)
	require.NoError(t, err)
	for _, member := range members {
		args.ACLAllow(member)
	}

	f, err := env.FM.Save(args)
	require.NoError(t, err)
	return f
}

// CreateBot BOTを必ず作成します
func (env *Env) CreateBot(t *testing.T, name string, creatorID uuid.UUID) *model.Bot {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	f := env.CreateFile(t, creatorID, uuid.Nil)
	b, err := env.Repository.CreateBot(name, "po", "totally a desc", f.GetID(), creatorID, model.BotModeHTTP, model.BotInactive, "https://example.com")
	require.NoError(t, err)
	return b
}

// CreateWebhook Webhookを必ず作成します
func (env *Env) CreateWebhook(t *testing.T, name string, creatorID, channelID uuid.UUID) model.Webhook {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	f := env.CreateFile(t, creatorID, uuid.Nil)
	w, err := env.Repository.CreateWebhook(name, "po", channelID, f.GetID(), creatorID, random.SecureAlphaNumeric(20))
	require.NoError(t, err)
	return w
}

// CreateOAuth2Client OAuth2クライアントを必ず作成します
func (env *Env) CreateOAuth2Client(t *testing.T, name string, creatorID uuid.UUID) *model.OAuth2Client {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	client := &model.OAuth2Client{
		ID:           random.SecureAlphaNumeric(36),
		Name:         name,
		Description:  "desc",
		Confidential: false,
		CreatorID:    creatorID,
		RedirectURI:  "https://example.com",
		Secret:       random.SecureAlphaNumeric(36),
		Scopes:       model.AccessScopes{"read": {}},
	}
	require.NoError(t, env.Repository.SaveClient(client))
	return client
}

// IssueToken OAuth2トークンを必ず発行します
func (env *Env) IssueToken(t *testing.T, client *model.OAuth2Client, userID uuid.UUID) *model.OAuth2Token {
	t.Helper()
	tok, err := env.Repository.IssueToken(client, userID, "https://example.com", model.AccessScopes{"read": {}}, 86400, false)
	require.NoError(t, err)
	return tok
}

// CreateClipFolder クリップフォルダを必ず作成します
func (env *Env) CreateClipFolder(t *testing.T, name, desc string, creatorID uuid.UUID) *model.ClipFolder {
	t.Helper()
	if name == rand {
		name = random.AlphaNumeric(20)
	}
	if desc == rand {
		desc = random.AlphaNumeric(20)
	}
	cf, err := env.Repository.CreateClipFolder(creatorID, name, desc)
	require.NoError(t, err)
	return cf
}

func getEnvOrDefault(env string, def string) string {
	s := os.Getenv(env)
	if len(s) == 0 {
		return def
	}
	return s
}
