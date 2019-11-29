package cmd

import (
	"cloud.google.com/go/profiler"
	"context"
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/fcm"
	"github.com/traPtitech/traQ/logging"
	"github.com/traPtitech/traQ/notification"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/router/sse"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var serveCommand = &cobra.Command{
	Use:   "serve",
	Short: "Serve traQ API",
	Run: func(cmd *cobra.Command, args []string) {
		versionAndRevision := fmt.Sprintf("%s.%s", Version, Revision)

		// Logger
		logger, err := logging.CreateNewLogger("traq", versionAndRevision)
		if err != nil {
			panic(err)
		}
		defer logger.Sync()

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

		// Realtime Service
		rt := realtime.NewService(hub)
		wss := ws.NewStreamer(hub, rt, logger.Named("ws"))
		sses := sse.NewStreamer(hub)

		// Notification Service
		notification.StartService(repo, hub, logger.Named("notification"), fcmClient, sses, wss, rt, viper.GetString("origin"))

		// HTTP Router
		e := router.Setup(&router.Config{
			Version:          Version,
			Revision:         Revision,
			AccessLogging:    viper.GetBool("accessLog.enabled"),
			Gzipped:          viper.GetBool("gzip"),
			ImageMagickPath:  viper.GetString("imagemagick.path"),
			AccessTokenExp:   viper.GetInt("oauth2.accessTokenExp"),
			IsRefreshEnabled: viper.GetBool("oauth2.isRefreshEnabled"),
			SkyWaySecretKey:  viper.GetString("skyway.secretKey"),
			Hub:              hub,
			Repository:       repo,
			RBAC:             r,
			WS:               wss,
			SSE:              sses,
			Realtime:         rt,
			RootLogger:       logger,
		})

		go func() {
			if err := e.Start(fmt.Sprintf(":%d", viper.GetInt("port"))); err != nil {
				logger.Info("shutting down the server")
			}
		}()

		logger.Info("traQ started")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sses.Dispose()
		wss.Close()
		if err := e.Shutdown(ctx); err != nil {
			logger.Warn("abnormal shutdown", zap.Error(err))
		}
		sessions.PurgeCache()
		logger.Info("traQ shutdown")
	},
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
