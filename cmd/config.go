package cmd

import (
	"cloud.google.com/go/profiler"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/service/counter"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/variable"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"image"
	"time"
)

// Config 設定
type Config struct {
	// DevMode 開発モードかどうか (default: false)
	DevMode bool `mapstructure:"dev" yaml:"dev"`
	// Pprof pprofを有効にするかどうか (default: false)
	Pprof bool `mapstructure:"pprof" yaml:"pprof"`

	// Origin サーバーオリジン (default: http://localhost:3000)
	Origin string `mapstructure:"origin" yaml:"origin"`
	// Port サーバーポート番号 (default: 3000)
	Port int `mapstructure:"port" yaml:"port"`
	// Gzip レスポンスのGZIP圧縮を有効にするかどうか (default: true)
	Gzip bool `mapstructure:"gzip" yaml:"gzip"`

	// AccessLog HTTPアクセスログ設定
	AccessLog struct {
		// Enabled 有効かどうか (default: true)
		Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	} `mapstructure:"accessLog" yaml:"accessLog"`

	// ImageMagick ImageMagick実行ファイルパス
	ImageMagick string `mapstructure:"imagemagick" yaml:"imagemagick"`

	// Imaging 画像処理設定
	Imaging struct {
		// MaxPixels 処理可能な最大画素数 (default: 2560*1600)
		MaxPixels int `mapstructure:"maxPixels" yaml:"maxPixels"`
		// Concurrency 処理並列数 (default: 1)
		Concurrency int `mapstructure:"concurrency" yaml:"concurrency"`
	} `mapstructure:"imaging" yaml:"imaging"`

	// MariaDB データベース接続設定
	MariaDB struct {
		// Host ホスト名 (default: 127.0.0.1)
		Host string `mapstructure:"host" yaml:"host"`
		// Port ポート番号 (default: 3306)
		Port int `mapstructure:"port" yaml:"port"`
		// Username ユーザー名 (default: root)
		Username string `mapstructure:"username" yaml:"username"`
		// Password パスワード (default: password)
		Password string `mapstructure:"password" yaml:"password"`
		// Database データベース名 (default: traq)
		Database string `mapstructure:"database" yaml:"database"`
		// Connection コネクション設定
		Connection struct {
			// MaxOpen 最大オープン接続数. 0は無制限 (default: 0)
			MaxOpen int `mapstructure:"maxOpen" yaml:"maxOpen"`
			// MaxIdle 最大アイドル接続数 (default: 2)
			MaxIdle int `mapstructure:"maxIdle" yaml:"maxIdle"`
			// LifeTime 待機接続維持時間. 0は無制限 (default: 0)
			LifeTime int `mapstructure:"lifetime" yaml:"lifetime"`
		} `mapstructure:"connection" yaml:"connection"`
	} `mapstructure:"mariadb" yaml:"mariadb"`

	// Storage ファイルストレージ設定
	Storage struct {
		// Type ストレージタイプ (default: local)
		// 	local: ローカルストレージ
		// 	swift: Swiftオブジェクトストレージ
		// 	memory: メモリストレージ
		Type string `mapstructure:"type" yaml:"type"`

		// Local ローカルストレージ設定
		Local struct {
			// Dir 保存先ディレクトリ (default: ./storage)
			Dir string `mapstructure:"dir" yaml:"dir"`
		} `mapstructure:"local" yaml:"local"`

		// Swift Swiftオブジェクトストレージ設定
		Swift struct {
			// UserName ユーザー名
			UserName string `mapstructure:"username" yaml:"username"`
			// APIKey APIキー(パスワード)
			APIKey string `mapstructure:"apiKey" yaml:"apiKey"`
			// TenantName テナント名
			TenantName string `mapstructure:"tenantName" yaml:"tenantName"`
			// TenantID テナントID
			TenantID string `mapstructure:"tenantId" yaml:"tenantId"`
			// Container コンテナ名
			Container string `mapstructure:"container" yaml:"container"`
			// AuthURL 認証エンドポイント
			AuthURL string `mapstructure:"authUrl" yaml:"authUrl"`
			// TempURLKey 一時URL発行キー
			TempURLKey string `mapstructure:"tempUrlKey" yaml:"tempUrlKey"`
			// CacheDir キャッシュディレクトリ
			CacheDir string `mapstructure:"cacheDir" yaml:"cacheDir"`
		} `mapstructure:"swift" yaml:"swift"`
	} `mapstructure:"storage" yaml:"storage"`

	// GCP Google Cloud Platform設定
	GCP struct {
		// ServiceAccount サービスアカウント設定
		ServiceAccount struct {
			// ProjectID Google Cloud Console プロジェクトID
			ProjectID string `mapstructure:"projectId" yaml:"projectId"`
			// File クレデンシャルファイル
			File string `mapstructure:"file" yaml:"file"`
		} `mapstructure:"serviceAccount" yaml:"serviceAccount"`

		// Stackdriver Stackdriver設定
		Stackdriver struct {
			// Profiler Stackdriver Profiler設定
			Profiler struct {
				// Enabled 有効かどうか
				Enabled bool `mapstructure:"enabled" yaml:"enabled"`
			} `mapstructure:"profiler" yaml:"profiler"`
		} `mapstructure:"stackdriver" yaml:"stackdriver"`
	} `mapstructure:"gcp" yaml:"gcp"`

	// Firebase Firebase設定
	Firebase struct {
		// ServiceAccount サービスアカウント設定
		ServiceAccount struct {
			// File クレデンシャルファイル
			File string `mapstructure:"file" yaml:"file"`
		} `mapstructure:"serviceAccount" yaml:"serviceAccount"`
	} `mapstructure:"firebase" yaml:"firebase"`

	// OAuth2 OAuth2認可サーバー設定
	OAuth2 struct {
		// IsRefreshEnabled リフレッシュトークンを有効にするかどうか (default: false)
		IsRefreshEnabled bool `mapstructure:"isRefreshEnabled" yaml:"isRefreshEnabled"`
		// AccessTokenExpire アクセストークン有効期間(秒) (default: 31536000)
		AccessTokenExpire int `mapstructure:"accessTokenExp" yaml:"accessTokenExp"`
	} `mapstructure:"oauth2" yaml:"oauth2"`

	// ExternalAuthentication 外部認証設定
	ExternalAuthentication struct {
		// Enabled 有効かどうか (default: false)
		Enabled  bool `mapstructure:"enabled" yaml:"enabled"`
		AuthPost struct {
			URL             string `mapstructure:"url" yaml:"url"`
			SuccessfulCode  int    `mapstructure:"successfulCode" yaml:"successfulCode"`
			FormUserNameKey string `mapstructure:"formUserNameKey" yaml:"formUserNameKey"`
			FormPasswordKey string `mapstructure:"formPasswordKey" yaml:"formPasswordKey"`
		} `mapstructure:"authPost" yaml:"authPost"`
	} `mapstructure:"externalAuthentication" yaml:"externalAuthentication"`

	// SkyWay SkyWay設定
	SkyWay struct {
		// SecretKey シークレットキー
		SecretKey string `mapstructure:"secretKey" yaml:"secretKey"`
	} `mapstructure:"skyway" yaml:"skyway"`

	// JWT JsonWebToken設定
	JWT struct {
		// Keys 鍵設定
		Keys struct {
			// Private ECDSA秘密鍵ファイル
			Private string `mapstructure:"private" yaml:"private"`
		} `mapstructure:"keys" yaml:"keys"`
	} `mapstructure:"jwt" yaml:"jwt"`

	// ExternalAuth 外部認証設定
	ExternalAuth struct {
		GitHub struct {
			ClientID     string `mapstructure:"clientId" yaml:"clientId"`
			ClientSecret string `mapstructure:"clientSecret" yaml:"clientSecret"`
			AllowSignUp  bool   `mapstructure:"allowSignUp" yaml:"allowSignUp"`
		} `mapstructure:"github" yaml:"github"`
		Google struct {
			ClientID     string `mapstructure:"clientId" yaml:"clientId"`
			ClientSecret string `mapstructure:"clientSecret" yaml:"clientSecret"`
			AllowSignUp  bool   `mapstructure:"allowSignUp" yaml:"allowSignUp"`
		} `mapstructure:"google" yaml:"google"`
		TraQ struct {
			Origin       string `mapstructure:"origin" yaml:"origin"`
			ClientID     string `mapstructure:"clientId" yaml:"clientId"`
			ClientSecret string `mapstructure:"clientSecret" yaml:"clientSecret"`
			AllowSignUp  bool   `mapstructure:"allowSignUp" yaml:"allowSignUp"`
		} `mapstructure:"traq" yaml:"traq"`
		OIDC struct {
			Issuer       string   `mapstructure:"issuer" yaml:"issuer"`
			ClientID     string   `mapstructure:"clientId" yaml:"clientId"`
			ClientSecret string   `mapstructure:"clientSecret" yaml:"clientSecret"`
			AllowSignUp  bool     `mapstructure:"allowSignUp" yaml:"allowSignUp"`
			Scopes       []string `mapstructure:"scopes" yaml:"scopes"`
		} `mapstructure:"oidc" yaml:"oidc"`
	} `mapstructure:"externalAuth" yaml:"externalAuth"`
}

// Configのデフォルト値設定
func init() {
	viper.SetDefault("dev", false)
	viper.SetDefault("pprof", false)
	viper.SetDefault("origin", "http://localhost:3000")
	viper.SetDefault("port", 3000)
	viper.SetDefault("gzip", true)
	viper.SetDefault("accessLog.enabled", true)
	viper.SetDefault("imagemagick", "")
	viper.SetDefault("imaging.maxPixels", 2560*1600)
	viper.SetDefault("imaging.concurrency", 1)
	viper.SetDefault("mariadb.host", "127.0.0.1")
	viper.SetDefault("mariadb.port", 3306)
	viper.SetDefault("mariadb.username", "root")
	viper.SetDefault("mariadb.password", "password")
	viper.SetDefault("mariadb.database", "traq")
	viper.SetDefault("mariadb.connection.maxOpen", 0)
	viper.SetDefault("mariadb.connection.maxIdle", 2)
	viper.SetDefault("mariadb.connection.lifetime", 0)
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.local.dir", "./storage")
	viper.SetDefault("storage.swift.username", "")
	viper.SetDefault("storage.swift.apiKey", "")
	viper.SetDefault("storage.swift.tenantName", "")
	viper.SetDefault("storage.swift.tenantId", "")
	viper.SetDefault("storage.swift.container", "")
	viper.SetDefault("storage.swift.authUrl", "")
	viper.SetDefault("storage.swift.tempUrlKey", "")
	viper.SetDefault("storage.swift.cacheDir", "")
	viper.SetDefault("gcp.serviceAccount.projectId", "")
	viper.SetDefault("gcp.serviceAccount.file", "")
	viper.SetDefault("gcp.stackdriver.profiler.enabled", false)
	viper.SetDefault("firebase.serviceAccount.file", "")
	viper.SetDefault("oauth2.isRefreshEnabled", false)
	viper.SetDefault("oauth2.accessTokenExp", 60*60*24*365)
	viper.SetDefault("externalAuthentication.enabled", false)
	viper.SetDefault("externalAuthentication.authPost.url", "")
	viper.SetDefault("externalAuthentication.authPost.successfulCode", 0)
	viper.SetDefault("externalAuthentication.authPost.formUserNameKey", "")
	viper.SetDefault("externalAuthentication.authPost.formPasswordKey", "")
	viper.SetDefault("externalAuth.github.clientId", "")
	viper.SetDefault("externalAuth.github.clientSecret", "")
	viper.SetDefault("externalAuth.github.allowSignUp", false)
	viper.SetDefault("externalAuth.google.clientId", "")
	viper.SetDefault("externalAuth.google.clientSecret", "")
	viper.SetDefault("externalAuth.google.allowSignUp", false)
	viper.SetDefault("externalAuth.traq.origin", "")
	viper.SetDefault("externalAuth.traq.clientId", "")
	viper.SetDefault("externalAuth.traq.clientSecret", "")
	viper.SetDefault("externalAuth.traq.allowSignUp", false)
	viper.SetDefault("externalAuth.oidc.issuer", "")
	viper.SetDefault("externalAuth.oidc.clientId", "")
	viper.SetDefault("externalAuth.oidc.clientSecret", "")
	viper.SetDefault("externalAuth.oidc.scopes", []string{})
	viper.SetDefault("externalAuth.oidc.allowSignUp", false)
	viper.SetDefault("skyway.secretKey", "")
	viper.SetDefault("jwt.keys.private", "")
}

func (c Config) getFileStorage() (storage.FileStorage, error) {
	switch c.Storage.Type {
	case "swift":
		return storage.NewSwiftFileStorage(
			c.Storage.Swift.Container,
			c.Storage.Swift.UserName,
			c.Storage.Swift.APIKey,
			c.Storage.Swift.TenantName,
			c.Storage.Swift.TenantID,
			c.Storage.Swift.AuthURL,
			c.Storage.Swift.TempURLKey,
			c.Storage.Swift.CacheDir,
		)
	case "composite":
		return storage.NewCompositeFileStorage(
			c.Storage.Local.Dir,
			c.Storage.Swift.Container,
			c.Storage.Swift.UserName,
			c.Storage.Swift.APIKey,
			c.Storage.Swift.TenantName,
			c.Storage.Swift.TenantID,
			c.Storage.Swift.AuthURL,
			c.Storage.Swift.TempURLKey,
			c.Storage.Swift.CacheDir,
		)
	case "memory":
		return storage.NewInMemoryFileStorage(), nil
	default:
		return storage.NewLocalFileStorage(c.Storage.Local.Dir), nil
	}
}

func (c Config) getDatabase() (*gorm.DB, error) {
	engine, err := gorm.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true",
		c.MariaDB.Username,
		c.MariaDB.Password,
		c.MariaDB.Host,
		c.MariaDB.Port,
		c.MariaDB.Database,
	))
	if err != nil {
		return nil, err
	}
	engine.DB().SetMaxOpenConns(c.MariaDB.Connection.MaxOpen)
	engine.DB().SetMaxIdleConns(c.MariaDB.Connection.MaxIdle)
	engine.DB().SetConnMaxLifetime(time.Duration(c.MariaDB.Connection.LifeTime) * time.Second)
	engine.BlockGlobalUpdate(true)
	engine.LogMode(c.DevMode)
	return engine.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"), nil
}

func initStackdriverProfiler(c *Config) error {
	return profiler.Start(profiler.Config{
		Service:        "traq",
		ServiceVersion: fmt.Sprintf("%s.%s", Version, Revision),
		ProjectID:      c.GCP.ServiceAccount.ProjectID,
	}, option.WithCredentialsFile(c.GCP.ServiceAccount.File))
}

func newFCMClientIfAvailable(repo repository.Repository, logger *zap.Logger, unreadCounter counter.UnreadMessageCounter, file variable.FirebaseCredentialsFilePathString) (fcm.Client, error) {
	if len(file) > 0 {
		return fcm.NewClientWithCredentialsFile(repo, logger, unreadCounter, file)
	}
	return fcm.NewNullClient(), nil
}

func provideServerOriginString(c *Config) variable.ServerOriginString {
	return variable.ServerOriginString(c.Origin)
}

func provideFirebaseCredentialsFilePathString(c *Config) variable.FirebaseCredentialsFilePathString {
	return variable.FirebaseCredentialsFilePathString(c.Firebase.ServiceAccount.File)
}

func provideImageProcessorConfig(c *Config) imaging.Config {
	return imaging.Config{
		MaxPixels:        c.Imaging.MaxPixels,
		Concurrency:      c.Imaging.Concurrency,
		ThumbnailMaxSize: image.Pt(360, 480),
		ImageMagickPath:  c.ImageMagick,
	}
}

func provideAuthGithubProviderConfig(c *Config) auth.GithubProviderConfig {
	return auth.GithubProviderConfig{
		ClientID:               c.ExternalAuth.GitHub.ClientID,
		ClientSecret:           c.ExternalAuth.GitHub.ClientSecret,
		RegisterUserIfNotFound: c.ExternalAuth.GitHub.AllowSignUp,
	}
}

func provideAuthGoogleProviderConfig(c *Config) auth.GoogleProviderConfig {
	return auth.GoogleProviderConfig{
		ClientID:               c.ExternalAuth.Google.ClientID,
		ClientSecret:           c.ExternalAuth.Google.ClientSecret,
		CallbackURL:            c.Origin + "/api/auth/google/callback",
		RegisterUserIfNotFound: c.ExternalAuth.Google.AllowSignUp,
	}
}

func provideAuthOIDCProviderConfig(c *Config) auth.OIDCProviderConfig {
	return auth.OIDCProviderConfig{
		Issuer:                 c.ExternalAuth.OIDC.Issuer,
		ClientID:               c.ExternalAuth.OIDC.ClientID,
		ClientSecret:           c.ExternalAuth.OIDC.ClientSecret,
		Scopes:                 c.ExternalAuth.OIDC.Scopes,
		CallbackURL:            c.Origin + "/api/auth/oidc/callback",
		RegisterUserIfNotFound: c.ExternalAuth.OIDC.AllowSignUp,
	}
}

func provideAuthTraQProviderConfig(c *Config) auth.TraQProviderConfig {
	return auth.TraQProviderConfig{
		Origin:                 c.ExternalAuth.TraQ.Origin,
		ClientID:               c.ExternalAuth.TraQ.ClientID,
		ClientSecret:           c.ExternalAuth.TraQ.ClientSecret,
		CallbackURL:            c.Origin + "/api/auth/traq/callback",
		RegisterUserIfNotFound: c.ExternalAuth.TraQ.AllowSignUp,
	}
}

func provideRouterExternalAuthConfig(c *Config) router.ExternalAuthConfig {
	return router.ExternalAuthConfig{
		GitHub: provideAuthGithubProviderConfig(c),
		Google: provideAuthGoogleProviderConfig(c),
		TraQ:   provideAuthTraQProviderConfig(c),
		OIDC:   provideAuthOIDCProviderConfig(c),
	}
}

func provideRouterConfig(c *Config) *router.Config {
	return &router.Config{
		Development:      c.DevMode,
		Version:          Version,
		Revision:         Revision,
		AccessLogging:    c.AccessLog.Enabled,
		Gzipped:          c.Gzip,
		AccessTokenExp:   c.OAuth2.AccessTokenExpire,
		IsRefreshEnabled: c.OAuth2.IsRefreshEnabled,
		SkyWaySecretKey:  c.SkyWay.SecretKey,
		ExternalAuth:     provideRouterExternalAuthConfig(c),
	}
}
