package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
)

const (
	TraQProviderName          = "traQ"
	traQAPIRequestErrorFormat = "traQ api request error: %w"
)

type TraQProvider struct {
	config    TraQProviderConfig
	repo      repository.Repository
	fm        file.Manager
	logger    *zap.Logger
	sessStore session.Store
	oa2       oauth2.Config
}

type TraQProviderConfig struct {
	ClientID               string
	ClientSecret           string
	CallbackURL            string
	Origin                 string
	RegisterUserIfNotFound bool
}

func (c TraQProviderConfig) Valid() bool {
	return len(c.ClientSecret) > 0 && len(c.ClientID) > 0 && len(c.CallbackURL) > 0 && len(c.Origin) > 0
}

type traqUserInfo struct {
	p           *TraQProvider
	t           *oauth2.Token
	id          string
	name        string
	displayName string
}

func (u *traqUserInfo) GetProviderName() string {
	return TraQProviderName
}

func (u *traqUserInfo) GetID() string {
	return u.id
}

func (u *traqUserInfo) GetRawName() string {
	return u.name
}

func (u *traqUserInfo) GetName() string {
	return u.name
}

func (u *traqUserInfo) GetDisplayName() string {
	return u.displayName
}

func (u *traqUserInfo) GetProfileImage() ([]byte, error) {
	resp, err := http.Get(u.p.config.Origin + "/api/v3/public/icon/" + u.name)
	if err != nil {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *traqUserInfo) IsLoginAllowedUser() bool {
	return true // TODO
}

func NewTraQProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config TraQProviderConfig) *TraQProvider {
	return &TraQProvider{
		repo:      repo,
		fm:        fm,
		config:    config,
		logger:    logger,
		sessStore: sessStore,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.CallbackURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  config.Origin + "/api/v3/oauth2/authorize",
				TokenURL: config.Origin + "/api/v3/oauth2/token",
			},
			Scopes: []string{"read"},
		},
	}
}

func (p *TraQProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(p.sessStore, &p.oa2)(c)
}

func (p *TraQProvider) CallbackHandler(c echo.Context) error {
	return defaultCallbackHandler(p, &p.oa2, p.repo, p.fm, p.sessStore, p.config.RegisterUserIfNotFound)(c)
}

func (p *TraQProvider) FetchUserInfo(t *oauth2.Token) (UserInfo, error) {
	var ui traqUserInfo
	ui.p = p
	ui.t = t

	c := p.oa2.Client(context.Background(), t)

	resp, err := c.Get(p.config.Origin + "/api/v3/users/me")
	if err != nil {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	var u struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}
	if err := json.ConfigFastest.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf(traQAPIRequestErrorFormat, err)
	}
	ui.id = u.ID
	ui.name = u.Name
	ui.displayName = u.Name

	return &ui, nil
}

func (p *TraQProvider) L() *zap.Logger {
	return p.logger
}
