package main

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
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
	"time"
)

func main() {
	// enable pprof http handler
	if len(config.PprofEnabled) > 0 {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Message Hub
	hub := hub.New()

	// Database
	engine, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true", config.DatabaseUserName, config.DatabasePassword, config.DatabaseHostName, config.DatabaseName))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	engine.DB().SetMaxOpenConns(75)

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
		if err := initData(repo); err != nil {
			panic(err)
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
		Issuer: config.TRAQOrigin,
	}
	if public, private := config.RS256PublicKeyFile, config.RS256PrivateKeyFile; private != "" && public != "" {
		err := oauth.LoadKeys(loadKeys(private, public))
		if err != nil {
			panic(err)
		}
	}

	// Firebase
	if len(config.FirebaseServiceAccountJSONFile) > 0 {
		if _, err := NewFCMManager(repo, hub); err != nil {
			panic(err)
		}
	}

	// Routing
	h := router.NewHandlers(oauth, r, repo, hub)
	e := echo.New()
	router.SetupRouting(e, h)
	router.LoadWebhookTemplate("static/webhook/*.tmpl")

	go func() {
		if err := e.Start(":" + config.Port); err != nil {
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

func getFileStorage() (storage.FileStorage, error) {
	if config.OSContainer == "" || config.OSUserName == "" || config.OSPassword == "" || config.OSAuthURL == "" {
		return storage.NewLocalFileStorage(config.LocalStorageDir), nil
	}
	return storage.NewCompositeFileStorage(config.LocalStorageDir, config.OSContainer, config.OSUserName, config.OSPassword, config.OSTenantName, config.OSTenantID, config.OSAuthURL)
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
