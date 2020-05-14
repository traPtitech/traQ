package cmd

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	rbac "github.com/traPtitech/traQ/rbac/impl"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/jwt"
	"github.com/traPtitech/traQ/utils/random"
	"go.uber.org/zap"
	"io/ioutil"
	"time"
)

// serveCommand サーバー起動コマンド
func serveCommand() *cobra.Command {
	var skipInitEmojis bool

	cmd := cobra.Command{
		Use:   "serve",
		Short: "Serve traQ API",
		Run: func(cmd *cobra.Command, args []string) {
			// Logger
			logger := getLogger()
			defer logger.Sync()

			logger.Info(fmt.Sprintf("traQ %s (revision %s)\n", Version, Revision))

			// Stackdriver Profiler
			if c.GCP.Stackdriver.Profiler.Enabled {
				if err := initStackdriverProfiler(c); err != nil {
					logger.Fatal("failed to setup Stackdriver Profiler", zap.Error(err))
				}
				logger.Info("stackdriver profiler started")
			}

			// Message Hub
			hub := hub.New()

			// Database
			logger.Info("connecting database...")
			engine, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			engine.SetLogger(gormzap.New(logger.Named("gorm")))
			defer engine.Close()
			logger.Info("database connection was established")

			// FileStorage
			logger.Info("checking file storage...")
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}
			logger.Info("file storage is ok")

			// Repository
			logger.Info("setting up repository...")
			repo, err := repository.NewGormRepository(engine, fs, hub, logger)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}
			logger.Info("repository was set up")

			// Repository Sync
			logger.Info("syncing repository...")
			init, err := repo.Sync()
			if err != nil {
				logger.Fatal("failed to sync repository", zap.Error(err))
			}
			logger.Info("repository was synced")

			// 初期化
			if init {
				logger.Info("data initializing...")

				// generalチャンネル作成
				if ch, err := repo.CreatePublicChannel("general", uuid.Nil, uuid.Nil); err == nil {
					logger.Info("#general was created", zap.Stringer("cid", ch.ID))
				} else {
					logger.Error("failed to init general channel", zap.Error(err))
				}

				// unicodeスタンプインストール
				if !skipInitEmojis {
					if err := installEmojis(repo, logger, false); err != nil {
						logger.Error("failed to install unicode emojis", zap.Error(err))
					}
				}

				logger.Info("data initialization finished")
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
				privRaw, pubRaw := random.GenerateECDSAKey()
				_ = jwt.SetupSigner(privRaw)
				logger.Warn("a temporary key for QRCode JWT was generated. This key is valid only during this running.", zap.String("public_key", string(pubRaw)))
			}

			// サーバー作成
			server, err := newServer(hub, engine, repo, logger, r, c)
			if err != nil {
				logger.Fatal("failed to create server", zap.Error(err))
			}

			go func() {
				if err := server.Start(fmt.Sprintf(":%d", c.Port)); err != nil {
					logger.Info("shutting down the server")
				}
			}()

			logger.Info("traQ started")
			waitSIGINT()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				logger.Warn("abnormal shutdown", zap.Error(err))
			}
			sessions.PurgeCache()
			logger.Info("traQ shutdown")
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&skipInitEmojis, "skip-init-emojis", false, "skip initializing Unicode Emoji stamps")

	return &cmd
}

type Server struct {
	L      *zap.Logger
	SS     *service.Services
	Router *echo.Echo
}

func (s *Server) Start(address string) error {
	return s.Router.Start(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.SS.SSE.Dispose()
	_ = s.SS.WS.Close()
	if s.SS.FCM != nil {
		s.SS.FCM.Close()
	}
	return s.Router.Shutdown(ctx)
}
