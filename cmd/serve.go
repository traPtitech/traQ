package cmd

import (
	"cloud.google.com/go/profiler"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/fcm"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/sse"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/jwt"
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
		// Logger
		logger := getLogger()
		defer logger.Sync()

		// Stackdriver Profiler
		if c.GCP.Stackdriver.Profiler.Enabled {
			err := profiler.Start(profiler.Config{
				Service:        "traq",
				ServiceVersion: fmt.Sprintf("%s.%s", Version, Revision),
				ProjectID:      c.GCP.ServiceAccount.ProjectID,
			}, option.WithCredentialsFile(c.GCP.ServiceAccount.File))
			if err != nil {
				logger.Fatal("failed to setup Stackdriver Profiler", zap.Error(err))
			}
			logger.Info("stackdriver profiler started")
		}

		// Message Hub
		hub := hub.New()

		// Database
		engine, err := c.getDatabase()
		if err != nil {
			logger.Fatal("failed to connect database", zap.Error(err))
		}
		engine.SetLogger(gormzap.New(logger.Named("gorm")))
		defer engine.Close()

		// FileStorage
		fs, err := c.getFileStorage()
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
			if dir := c.InitDataDir; len(dir) > 0 {
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
		if f := c.Firebase.ServiceAccount.File; len(f) > 0 {
			fcmClient, err = fcm.NewClient(repo, logger.Named("fcm"), option.WithCredentialsFile(f))
			if err != nil {
				logger.Fatal("failed to setup firebase", zap.Error(err))
			}
		}

		// Bot Processor
		bot.NewProcessor(repo, hub, logger.Named("bot_processor"))

		// JWT for QRCode
		if priv := c.JWT.Keys.Private; priv != "" {
			privRaw, err := ioutil.ReadFile(priv)
			if err != nil {
				logger.Fatal("failed to read jwt private key", zap.Error(err))
			}
			if err := jwt.SetupSigner(privRaw); err != nil {
				logger.Fatal("failed to setup signer", zap.Error(err))
			}
		} else {
			// 一時鍵を発行
			priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			ecder, _ := x509.MarshalECPrivateKey(priv)
			ecderpub, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
			privRaw := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecder})
			pubRaw := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: ecderpub})
			_ = jwt.SetupSigner(privRaw)
			logger.Warn("a temporary key for QRCode JWT was generated. This key is valid only during this running.", zap.String("public_key", string(pubRaw)))
		}

		// Realtime Service
		rt := realtime.NewService(hub)
		wss := ws.NewStreamer(hub, rt, logger.Named("ws"))
		sses := sse.NewStreamer(hub)

		// Notification Service
		notification.StartService(repo, hub, logger.Named("notification"), fcmClient, sses, wss, rt, c.Origin)

		// HTTP Router
		e := router.Setup(&router.Config{
			Development:      c.DevMode,
			Version:          Version,
			Revision:         Revision,
			AccessLogging:    c.AccessLog.Enabled,
			Gzipped:          c.Gzip,
			ImageMagickPath:  c.ImageMagick,
			AccessTokenExp:   c.OAuth2.AccessTokenExpire,
			IsRefreshEnabled: c.OAuth2.IsRefreshEnabled,
			SkyWaySecretKey:  c.SkyWay.SecretKey,
			Hub:              hub,
			Repository:       repo,
			RBAC:             r,
			WS:               wss,
			SSE:              sses,
			Realtime:         rt,
			RootLogger:       logger,
		})

		go func() {
			if err := e.Start(fmt.Sprintf(":%d", c.Port)); err != nil {
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
