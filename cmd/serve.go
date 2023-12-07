package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/service"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/jwt"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/twemoji"
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

			logger.Info(fmt.Sprintf("traQ %s (revision %s)", Version, Revision))

			// Stackdriver Profiler
			if c.GCP.Stackdriver.Profiler.Enabled {
				if err := initStackdriverProfiler(&c); err != nil {
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
			engine.Logger = gormzap.New(logger.Named("gorm"))
			db, err := engine.DB()
			if err != nil {
				logger.Fatal("failed to get *sql.DB", zap.Error(err))
			}
			defer db.Close()
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
			repo, init, err := gorm.NewGormRepository(engine, hub, logger, true)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}
			logger.Info("repository was set up")

			// JWT for QRCode
			if priv := c.JWT.Keys.Private; priv != "" {
				privRaw, err := os.ReadFile(priv)
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
			server, err := newServer(hub, engine, repo, fs, logger, &c)
			if err != nil {
				logger.Fatal("failed to create server", zap.Error(err))
			}

			// 初期化
			if init {
				logger.Info("data initializing...")

				// システムユーザーロール投入
				if err := repo.CreateUserRoles(role.SystemRoleModels()...); err != nil {
					logger.Fatal("failed to init system user roles", zap.Error(err))
				}
				if err := server.SS.RBAC.Reload(); err != nil {
					logger.Fatal("failed to reload rbac", zap.Error(err))
				}

				// 管理者ユーザーの作成
				fid, err := file.GenerateIconFile(server.SS.FileManager, "traq")
				if err != nil {
					logger.Fatal("failed to generate icon file", zap.Error(err))
				}
				u, err := repo.CreateUser(repository.CreateUserArgs{
					Name:       "traq",
					Password:   "traq",
					Role:       role.Admin,
					IconFileID: fid,
				})
				if err == nil {
					logger.Info("traq user was created", zap.Stringer("uid", u.GetID()))
				} else {
					logger.Fatal("failed to init admin user", zap.Error(err))
				}

				// generalチャンネル作成
				if ch, err := server.SS.ChannelManager.CreatePublicChannel("general", uuid.Nil, uuid.Nil); err == nil {
					logger.Info("#general was created", zap.Stringer("cid", ch.ID))
				} else {
					logger.Error("failed to init general channel", zap.Error(err))
				}

				// unicodeスタンプインストール
				if !skipInitEmojis {
					if err := twemoji.Install(repo, server.SS.FileManager, logger, false); err != nil {
						logger.Error("failed to install unicode emojis", zap.Error(err))
					}
				}

				logger.Info("data initialization finished")
			}

			go func() {
				if err := server.Start(fmt.Sprintf(":%d", c.Port)); err != nil {
					logger.Info("shutting down the server")
				}
			}()

			logger.Info("traQ started")
			waitSIGINT()
			logger.Info("traQ shutting down...")

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.ShutdownTimeout)*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				logger.Warn("abnormal shutdown", zap.Error(err))
			}
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
	Hub    *hub.Hub
	Repo   repository.Repository
}

func (s *Server) Start(address string) error {
	go func() {
		// TODO 適切なパッケージに移動させる
		sub := s.Hub.Subscribe(10, event.UserOffline)
		for ev := range sub.Receiver {
			userID := ev.Fields["user_id"].(uuid.UUID)
			datetime := ev.Fields["datetime"].(time.Time)
			_ = s.Repo.UpdateUser(userID, repository.UpdateUserArgs{LastOnline: optional.From(datetime)})
		}
	}()
	s.SS.StampThrottler.Start()
	return s.Router.Start(address)
}

func (s *Server) Shutdown(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		err := s.Router.Shutdown(ctx)
		s.L.Info("Router shutdown")
		return err
	})
	eg.Go(func() error {
		err := s.SS.WS.Close()
		s.L.Info("WebSocket shutdown")
		return err
	})
	eg.Go(func() error {
		err := s.SS.BotWS.Close()
		s.L.Info("Bot WebSocket shutdown")
		return err
	})
	eg.Go(func() error {
		err := s.SS.BOT.Shutdown(ctx)
		s.L.Info("Bot shutdown")
		return err
	})
	eg.Go(func() error {
		err := s.SS.OGP.Shutdown()
		s.L.Info("OGP shutdown")
		return err
	})
	eg.Go(func() error {
		s.SS.FCM.Close()
		s.L.Info("FCM shutdown")
		return nil
	})
	eg.Go(func() error {
		s.SS.ChannelManager.Wait()
		s.L.Info("Channel manager shutdown")
		return nil
	})
	eg.Go(func() error {
		err := s.SS.MessageManager.Wait(ctx)
		s.L.Info("Message manager shutdown")
		return err
	})
	return eg.Wait()
}
