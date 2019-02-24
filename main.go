package main

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/oauth2/impl"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	repoimpl "github.com/traPtitech/traQ/repository/impl"
	"github.com/traPtitech/traQ/router"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils/storage"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	// set default config values
	viper.SetDefault("origin", "http://localhost:3000")
	viper.SetDefault("port", 3000)

	viper.SetDefault("pprof", false)

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

	// read config
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TRAQ")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal(err)
		}
	}

	// enable pprof http handler
	if viper.GetBool("pprof") {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Message Hub
	hub := hub.New()

	// Database
	engine, err := getDatabase()
	if err != nil {
		panic(err)
	}
	defer engine.Close()

	// FileStorage
	fs, err := getFileStorage()
	if err != nil {
		panic(err)
	}

	// Repository
	repo, err := repoimpl.NewRepositoryImpl(engine, fs, hub)
	if err != nil {
		panic(err)
	}
	if init, err := repo.Sync(); err != nil {
		panic(err)
	} else if init { // 初期化
		if dir := viper.GetString("initDataDir"); len(dir) > 0 {
			if err := initData(repo, dir); err != nil {
				panic(err)
			}
		}
	}

	// SessionStore
	sessionStore, err := sessions.NewGORMStore(engine)
	if err != nil {
		panic(err)
	}
	sessions.SetStore(sessionStore)

	// Init Role-Based Access Controller
	rbacStore, err := rbac.NewDefaultStore(engine)
	if err != nil {
		panic(err)
	}
	r, err := rbac.New(rbacStore)
	if err != nil {
		panic(err)
	}
	role.SetRole(r)

	// oauth2 handler
	oauth2Store, err := impl.NewDefaultStore(engine)
	if err != nil {
		panic(err)
	}
	oauth := &oauth2.Handler{
		Store:                oauth2Store,
		AccessTokenExp:       60 * 60 * 24 * 365, //1年
		AuthorizationCodeExp: 60 * 5,             //5分
		IsRefreshEnabled:     false,
		UserAuthenticator: func(id, pw string) (uuid.UUID, error) {
			user, err := repo.GetUserByName(id)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return uuid.Nil, oauth2.ErrUserIDOrPasswordWrong
				default:
					return uuid.Nil, err
				}
			}

			err = model.AuthenticateUser(user, pw)
			switch err {
			case model.ErrUserWrongIDOrPassword, model.ErrUserBotTryLogin:
				err = oauth2.ErrUserIDOrPasswordWrong
			}
			return user.ID, err
		},
		UserInfoGetter: func(uid uuid.UUID) (oauth2.UserInfo, error) {
			u, err := repo.GetUser(uid)
			if err == repository.ErrNotFound {
				return nil, oauth2.ErrUserIDOrPasswordWrong
			}
			return u, err
		},
		Issuer: viper.GetString("origin"),
	}
	if viper.IsSet("key.rs256Public") && viper.IsSet("key.rs256Private") {
		err := oauth.LoadKeys(loadKeys(viper.GetString("key.rs256Private"), viper.GetString("key.rs256Public")))
		if err != nil {
			panic(err)
		}
	}

	// Firebase
	if f := viper.GetString("firebase.serviceAccount.file"); len(f) > 0 {
		if _, err := NewFCMManager(repo, hub, f, viper.GetString("origin")); err != nil {
			panic(err)
		}
	}

	// Routing
	h := router.NewHandlers(oauth, r, repo, hub, viper.GetString("imagemagick.path"))
	e := echo.New()
	router.SetupRouting(e, h)
	router.LoadWebhookTemplate("static/webhook/*.tmpl")

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", viper.GetInt("port"))); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Error(err)
	}
	sessions.PurgeCache()
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

func loadKeys(private, public string) ([]byte, []byte) {
	prk, err := ioutil.ReadFile(private)
	if err != nil {
		panic(err)
	}
	puk, err := ioutil.ReadFile(public)
	if err != nil {
		panic(err)
	}
	return prk, puk
}
