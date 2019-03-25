package main

import (
	"cloud.google.com/go/profiler"
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/karixtech/zapdriver"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/role"
	repoimpl "github.com/traPtitech/traQ/repository/impl"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
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
	zc := &zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding:         "json",
		EncoderConfig:    zapdriver.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := zc.Build(zapdriver.WrapCoreWithConfig(zapdriver.DriverConfig{ReportAllErrors: true, ServiceName: "traq"}))
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
		go func() { _ = http.ListenAndServe("localhost:6060", nil) }()
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
	repo, err := repoimpl.NewRepositoryImpl(engine, fs, hub)
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

	if viper.GetBool("generateThumbnailOnStartUp") {
		var files []uuid.UUID
		if err := engine.Model(&model.File{}).Where("has_thumbnail = false").Pluck("id", &files).Error; err != nil {
			logger.Warn("failed to fetch no thumbnail files", zap.Error(err))
		}
		for _, v := range files {
			_, _ = repo.RegenerateThumbnail(v)
		}
	}

	// SessionStore
	sessionStore, err := sessions.NewGORMStore(engine)
	if err != nil {
		logger.Fatal("failed to setup session store", zap.Error(err))
	}
	sessions.SetStore(sessionStore)

	// Init Role-Based Access Controller
	rbacStore, err := rbac.NewDefaultStore(engine)
	if err != nil {
		logger.Fatal("failed to setup rbac store", zap.Error(err))
	}
	r, err := rbac.New(rbacStore)
	if err != nil {
		logger.Fatal("failed to init rbac", zap.Error(err))
	}
	role.SetRole(r)

	// Firebase
	if f := viper.GetString("firebase.serviceAccount.file"); len(f) > 0 {
		if _, err := NewFCMManager(repo, hub, logger.Named("firebase"), f, viper.GetString("origin")); err != nil {
			logger.Fatal("failed to setup firebase", zap.Error(err))
		}
	}

	// Bot Processor
	bot.NewProcessor(repo, hub, logger.Named("bot_processor"))

	// Routing
	h := router.NewHandlers(r, repo, hub, logger.Named("router"), router.HandlerConfig{
		ImageMagickPath:  viper.GetString("imagemagick.path"),
		AccessTokenExp:   viper.GetInt("oauth2.accessTokenExp"),
		IsRefreshEnabled: viper.GetBool("oauth2.isRefreshEnabled"),
	})
	e := echo.New()
	if viper.GetBool("accessLog.enabled") {
		alog := logger.Named("access_log")
		exHeartbeat := viper.GetBool("accessLog.excludesHeartbeat")
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				if exHeartbeat && strings.HasPrefix(c.Path(), "/api/1.0/heartbeat") {
					return next(c)
				}

				start := time.Now()
				if err := next(c); err != nil {
					c.Error(err)
				}
				stop := time.Now()

				req := c.Request()
				res := c.Response()
				alog.Info("", zap.String("logging.googleapis.com/trace", router.GetTraceID(c)), zapdriver.HTTP(&zapdriver.HTTPPayload{
					RequestMethod: req.Method,
					Status:        res.Status,
					UserAgent:     req.UserAgent(),
					RemoteIP:      c.RealIP(),
					Referer:       req.Referer(),
					Protocol:      req.Proto,
					RequestURL:    req.URL.String(),
					RequestSize:   req.Header.Get(echo.HeaderContentLength),
					ResponseSize:  strconv.FormatInt(res.Size, 10),
					Latency:       strconv.FormatFloat(stop.Sub(start).Seconds(), 'f', 9, 64) + "s",
				}))
				return nil
			}
		})
	}
	e.Use(router.AddHeadersMiddleware(map[string]string{"X-TRAQ-VERSION": versionAndRevision}))
	e.HideBanner = true
	e.HidePort = true
	router.SetupRouting(e, h)
	router.LoadWebhookTemplate("static/webhook/*.tmpl")

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", viper.GetInt("port"))); err != nil {
			logger.Info("shutting down the server")
		}
	}()

	logger.Info("traQ started", zap.String("version", versionAndRevision))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Warn("abnormal shutdown", zap.Error(err))
	}
	sessions.PurgeCache()
}

func setDefaultConfigs() {
	viper.SetDefault("origin", "http://localhost:3000")
	viper.SetDefault("port", 3000)
	viper.SetDefault("accessLog.enabled", true)
	viper.SetDefault("accessLog.excludesHeartbeat", true)

	viper.SetDefault("pprof", false)

	viper.SetDefault("generateThumbnailOnStartUp", false)

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
	engine.LogMode(false)
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
		)
	case "memory":
		return storage.NewInMemoryFileStorage(), nil
	default:
		return storage.NewLocalFileStorage(viper.GetString("storage.local.dir")), nil
	}
}
