package main

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/sessions"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/external/storage"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/oauth2/impl"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/router"
)

func main() {
	// enable pprof http handler
	if len(config.PprofEnabled) > 0 {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Database
	engine, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true", config.DatabaseUserName, config.DatabasePassword, config.DatabaseHostName, config.DatabaseName))
	if err != nil {
		panic(err)
	}
	defer engine.Close()
	model.SetGORMEngine(engine)

	if init, err := model.Sync(); err != nil {
		panic(err)
	} else if init { // 初期化
		if err := initData(); err != nil {
			panic(err)
		}
	}

	sessionStore, err := sessions.NewGORMStore(engine)
	if err != nil {
		panic(err)
	}
	sessions.SetStore(sessionStore)

	// ObjectStorage
	if err := setSwiftFileManagerAsDefault(
		config.OSContainer,
		config.OSUserName,
		config.OSPassword,
		config.OSTenantName, //v2のみ
		config.OSTenantID,   //v2のみ
		config.OSAuthURL,
	); err != nil {
		panic(err)
	}

	// Init Caches
	if err := model.InitCache(); err != nil {
		panic(err)
	}

	// Init Role-Based Access Controller
	r, err := rbac.New(&model.RBACOverrideStore{})
	if err != nil {
		panic(err)
	}
	role.SetRole(r)

	// oauth2 handler
	oauth := &oauth2.Handler{
		Store:                &impl.DefaultStore{},
		AccessTokenExp:       60 * 60 * 24 * 365, //1年
		AuthorizationCodeExp: 60 * 5,             //5分
		IsRefreshEnabled:     false,
		UserAuthenticator: func(id, pw string) (uuid.UUID, error) {
			user, err := model.GetUserByName(id)
			if err != nil {
				switch err {
				case model.ErrNotFound:
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
			return uuid.FromStringOrNil(user.ID), err
		},
		UserInfoGetter: func(uid uuid.UUID) (oauth2.UserInfo, error) {
			u, err := model.GetUser(uid)
			if err == model.ErrNotFound {
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

	// event handler
	if len(config.FirebaseServiceAccountJSONFile) > 0 {
		fcm := &event.FCMManager{}
		if err := fcm.Init(); err != nil {
			panic(err)
		}
		event.AddListener(fcm)
	}

	h := &router.Handlers{
		Bot:    event.NewBotProcessor(oauth),
		OAuth2: oauth,
		RBAC:   r,
	}
	event.AddListener(h.Bot)

	e := echo.New()
	router.SetupRouting(e, h)
	router.LoadWebhookTemplate("static/webhook/*.tmpl")

	// init heartbeat
	model.OnUserOnlineStateChanged = func(id uuid.UUID, online bool) {
		if online {
			go event.Emit(event.UserOnline, &event.UserEvent{ID: id})
		} else {
			go event.Emit(event.UserOffline, &event.UserEvent{ID: id})
		}
	}
	model.HeartbeatStart()

	go func() {
		if err := e.Start(":" + config.Port); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Error(err)
	}
	sessions.PurgeCache()
}

func setSwiftFileManagerAsDefault(container, userName, apiKey, tenant, tenantID, authURL string) error {
	if container == "" || userName == "" || apiKey == "" || authURL == "" {
		return nil
	}
	m, err := storage.NewSwiftFileManager(container, userName, apiKey, tenant, tenantID, authURL, false) //TODO リダイレクトをオンにする
	if err != nil {
		return err
	}
	model.SetFileManager("", m)
	return nil
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
