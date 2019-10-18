package main

import (
	"context"
	"fmt"
	"github.com/traPtitech/traQ/fcm"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/realtime"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/profiler"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/logging"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

var (
	version  = "UNKNOWN"
	revision = "UNKNOWN"
)

func main() {
	versionAndRevision := fmt.Sprintf("%s.%s", version, revision)

	// set default config values
	setDefaultConfigs()

	// Logger
	logger, err := logging.CreateNewLogger("traq", versionAndRevision)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// read config
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TRAQ")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logger.Fatal("failed to read config file", zap.Error(err))
		}
	}

	// enable pprof http handler
	if viper.GetBool("pprof") {
		go func() { _ = http.ListenAndServe("0.0.0.0:6060", nil) }()
	}

	// Stackdriver Profiler
	if viper.GetBool("gcp.stackdriver.profiler.enabled") {
		err := profiler.Start(profiler.Config{
			Service:        "traq",
			ServiceVersion: versionAndRevision,
			ProjectID:      viper.GetString("gcp.serviceAccount.projectId"),
		}, option.WithCredentialsFile(viper.GetString("gcp.serviceAccount.file")))
		if err != nil {
			logger.Fatal("failed to setup Stackdriver Profiler", zap.Error(err))
		}
	}

	// Message Hub
	hub := hub.New()

	// Database
	engine, err := getDatabase()
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}
	defer engine.Close()

	// FileStorage
	fs, err := getFileStorage()
	if err != nil {
		logger.Fatal("failed to setup file storage", zap.Error(err))
	}

	// Repository
	repo, err := repository.NewGormRepository(engine, fs, hub, logger.Named("repository"))
	if err != nil {
		logger.Fatal("failed to initialize repository", zap.Error(err))
	}
	if init, err := repo.Sync(); err != nil {
		logger.Fatal("failed to sync repository", zap.Error(err))
	} else if init { // 初期化
		if dir := viper.GetString("initDataDir"); len(dir) > 0 {
			if err := initData(repo, dir); err != nil {
				logger.Fatal("failed to init data", zap.Error(err))
			}
		}
	}

	// SessionStore
	sessionStore, err := sessions.NewGORMStore(engine)
	if err != nil {
		logger.Fatal("failed to setup session store", zap.Error(err))
	}
	sessions.SetStore(sessionStore)

	// Role-Based Access Controller
	r, err := rbac.New(repo)
	if err != nil {
		logger.Fatal("failed to init rbac", zap.Error(err))
	}

	// Firebase
	var fcmClient *fcm.Client
	if f := viper.GetString("firebase.serviceAccount.file"); len(f) > 0 {
		fcmClient, err = fcm.NewClient(repo, logger.Named("fcm"), option.WithCredentialsFile(f))
		if err != nil {
			logger.Fatal("failed to setup firebase", zap.Error(err))
		}
	}

	// Bot Processor
	bot.NewProcessor(repo, hub, logger.Named("bot_processor"))

	// JWT for QRCode
	pubRaw, err := ioutil.ReadFile(viper.GetString("jwt.keys.public"))
	if err != nil {
		logger.Fatal("failed to read jwt public key", zap.Error(err))
	}
	privRaw, err := ioutil.ReadFile(viper.GetString("jwt.keys.private"))
	if err != nil {
		logger.Fatal("failed to read jwt private key", zap.Error(err))
	}
	if err := utils.SetupSigner(pubRaw, privRaw); err != nil {
		logger.Fatal("failed to setup signer", zap.Error(err))
	}

	// Realtime Manager
	rt := realtime.NewManager(hub)

	// Routing
	h := router.NewHandlers(r, repo, hub, logger.Named("router"), rt, router.HandlerConfig{
		ImageMagickPath:  viper.GetString("imagemagick.path"),
		AccessTokenExp:   viper.GetInt("oauth2.accessTokenExp"),
		IsRefreshEnabled: viper.GetBool("oauth2.isRefreshEnabled"),
		SkyWaySecretKey:  viper.GetString("skyway.secretKey"),
	})
	e := echo.New()
	if viper.GetBool("accessLog.enabled") {
		e.Use(router.AccessLoggingMiddleware(logger.Named("access_log"), viper.GetBool("accessLog.excludesHeartbeat")))
	}
	if viper.GetBool("gzip") {
		e.Use(router.Gzip())
	}
	e.Use(router.AddHeadersMiddleware(map[string]string{"X-TRAQ-VERSION": versionAndRevision}))
	e.HideBanner = true
	e.HidePort = true
	router.SetupRouting(e, h)
	router.LoadWebhookTemplate("static/webhook/*.tmpl")

	// Notification Service
	notification.StartService(repo, hub, logger.Named("notification"), fcmClient, h.SSE, rt, viper.GetString("origin"))

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", viper.GetInt("port"))); err != nil {
			logger.Info("shutting down the server")
		}
	}()

	logger.Info("traQ started", zap.String("version", versionAndRevision))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	h.SSE.Dispose()
	if err := e.Shutdown(ctx); err != nil {
		logger.Warn("abnormal shutdown", zap.Error(err))
	}
	sessions.PurgeCache()
	logger.Info("traQ shutdown")
}

func setDefaultConfigs() {
	viper.SetDefault("origin", "http://localhost:3000")
	viper.SetDefault("port", 3000)
	viper.SetDefault("gzip", true)
	viper.SetDefault("accessLog.enabled", true)
	viper.SetDefault("accessLog.excludesHeartbeat", true)

	viper.SetDefault("pprof", false)
	viper.SetDefault("gormLogMode", false)

	viper.SetDefault("externalAuthentication.enabled", false)

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

	viper.SetDefault("gcp.stackdriver.profiler.enabled", false)

	viper.SetDefault("oauth2.isRefreshEnabled", false)
	viper.SetDefault("oauth2.accessTokenExp", 60*60*24*365)

	viper.SetDefault("jwt.keys.public", "./keys/ec_pub.pem")
	viper.SetDefault("jwt.keys.private", "./keys/ec.pem")

	viper.SetDefault("skyway.secretKey", "")
}

func getDatabase() (*gorm.DB, error) {
	engine, err := gorm.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true",
		viper.GetString("mariadb.username"),
		viper.GetString("mariadb.password"),
		viper.GetString("mariadb.host"),
		viper.GetInt("mariadb.port"),
		viper.GetString("mariadb.database"),
	))
	if err != nil {
		return nil, err
	}
	engine.DB().SetMaxOpenConns(viper.GetInt("mariadb.connection.maxOpen"))
	engine.DB().SetMaxIdleConns(viper.GetInt("mariadb.connection.maxIdle"))
	engine.DB().SetConnMaxLifetime(time.Duration(viper.GetInt("mariadb.connection.lifetime")) * time.Second)
	engine.LogMode(viper.GetBool("gormLogMode"))
	return engine, nil
}

func getFileStorage() (storage.FileStorage, error) {
	switch viper.GetString("storage.type") {
	case "swift":
		return storage.NewSwiftFileStorage(
			viper.GetString("storage.swift.container"),
			viper.GetString("storage.swift.username"),
			viper.GetString("storage.swift.apiKey"),
			viper.GetString("storage.swift.tenantName"),
			viper.GetString("storage.swift.tenantId"),
			viper.GetString("storage.swift.authUrl"),
			viper.GetString("storage.swift.tempUrlKey"),
			viper.GetString("storage.swift.cacheDir"),
		)
	case "composite":
		return storage.NewCompositeFileStorage(
			viper.GetString("storage.local.dir"),
			viper.GetString("storage.swift.container"),
			viper.GetString("storage.swift.username"),
			viper.GetString("storage.swift.apiKey"),
			viper.GetString("storage.swift.tenantName"),
			viper.GetString("storage.swift.tenantId"),
			viper.GetString("storage.swift.authUrl"),
			viper.GetString("storage.swift.tempUrlKey"),
			viper.GetString("storage.swift.cacheDir"),
		)
	case "memory":
		return storage.NewInMemoryFileStorage(), nil
	default:
		return storage.NewLocalFileStorage(viper.GetString("storage.local.dir")), nil
	}
}
