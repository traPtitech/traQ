package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
	google "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
)

const (
	GoogleProviderName          = "google"
	googleAPIRequestErrorFormat = "google api request error: %w"
)

type GoogleProvider struct {
	config    GoogleProviderConfig
	repo      repository.Repository
	fm        file.Manager
	logger    *zap.Logger
	sessStore session.Store
	oa2       oauth2.Config
}

type GoogleProviderConfig struct {
	ClientID               string
	ClientSecret           string
	CallbackURL            string
	RegisterUserIfNotFound bool
}

func (c GoogleProviderConfig) Valid() bool {
	return len(c.ClientSecret) > 0 && len(c.ClientID) > 0 && len(c.CallbackURL) > 0
}

type googleUserInfo struct {
	p               *GoogleProvider
	t               *oauth2.Token
	id              string
	displayName     string
	email           string
	profileImageURL string
}

func (u *googleUserInfo) GetProviderName() string {
	return GoogleProviderName
}

func (u *googleUserInfo) GetID() string {
	return u.id
}

func (u *googleUserInfo) GetRawName() string {
	return u.email
}

func (u *googleUserInfo) GetName() string {
	return strings.ReplaceAll(strings.Split(u.email, "@")[0], ".", "_")
}

func (u *googleUserInfo) GetDisplayName() string {
	if s := utf8string.NewString(u.displayName); s.RuneCount() > 64 {
		return s.Slice(0, 64)
	}
	return u.displayName
}

func (u *googleUserInfo) GetProfileImage() ([]byte, error) {
	if len(u.profileImageURL) == 0 {
		return nil, nil
	}
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(u.profileImageURL)
	if err != nil {
		return nil, fmt.Errorf(googleAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(googleAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(googleAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *googleUserInfo) IsLoginAllowedUser() bool {
	return true // TODO
}

func NewGoogleProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config GoogleProviderConfig) *GoogleProvider {
	return &GoogleProvider{
		repo:      repo,
		fm:        fm,
		config:    config,
		logger:    logger,
		sessStore: sessStore,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.CallbackURL,
			Endpoint:     googleOAuth2.Endpoint,
			Scopes:       []string{"profile", "email"},
		},
	}
}

func (p *GoogleProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(p.sessStore, &p.oa2)(c)
}

func (p *GoogleProvider) CallbackHandler(c echo.Context) error {
	return defaultCallbackHandler(p, &p.oa2, p.repo, p.fm, p.sessStore, p.config.RegisterUserIfNotFound)(c)
}

func (p *GoogleProvider) FetchUserInfo(t *oauth2.Token) (UserInfo, error) {
	var ui googleUserInfo
	ui.p = p
	ui.t = t

	c := p.oa2.Client(context.Background(), t)
	googleService, err := google.NewService(context.Background(), option.WithHTTPClient(c))
	if err != nil {
		return nil, fmt.Errorf(googleAPIRequestErrorFormat, err)
	}
	u, err := googleService.Userinfo.Get().Do()
	if err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}

	ui.id = u.Id
	ui.email = u.Email
	ui.displayName = u.Name
	ui.profileImageURL = u.Picture

	return &ui, nil
}

func (p *GoogleProvider) L() *zap.Logger {
	return p.logger
}
