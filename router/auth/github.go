package auth

import (
	"context"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	GithubProviderName          = "github"
	githubProfileURL            = "https://api.github.com/user"
	githubAPIRequestErrorFormat = "github api request error: %w"
)

type GithubProvider struct {
	config    GithubProviderConfig
	repo      repository.Repository
	fm        file.Manager
	logger    *zap.Logger
	sessStore session.Store
	oa2       oauth2.Config
}

type GithubProviderConfig struct {
	ClientID               string
	ClientSecret           string
	RegisterUserIfNotFound bool
}

func (c GithubProviderConfig) Valid() bool {
	return len(c.ClientSecret) > 0 && len(c.ClientID) > 0
}

type githubUserInfo struct {
	p               *GithubProvider
	t               *oauth2.Token
	id              int
	displayName     string
	name            string
	profileImageURL string
}

func (u *githubUserInfo) GetProviderName() string {
	return GithubProviderName
}

func (u *githubUserInfo) GetID() string {
	return strconv.Itoa(u.id)
}

func (u *githubUserInfo) GetRawName() string {
	return u.name
}

func (u *githubUserInfo) GetName() string {
	if s := utf8string.NewString(u.name); s.RuneCount() > 32 {
		return s.Slice(0, 32)
	}
	return u.name
}

func (u *githubUserInfo) GetDisplayName() string {
	if s := utf8string.NewString(u.displayName); s.RuneCount() > 64 {
		return s.Slice(0, 64)
	}
	return u.displayName
}

func (u *githubUserInfo) GetProfileImage() ([]byte, error) {
	if len(u.profileImageURL) == 0 {
		return nil, nil
	}
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(u.profileImageURL)
	if err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *githubUserInfo) IsLoginAllowedUser() bool {
	return true // TODO
}

func NewGithubProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config GithubProviderConfig) *GithubProvider {
	return &GithubProvider{
		repo:      repo,
		fm:        fm,
		config:    config,
		logger:    logger,
		sessStore: sessStore,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       []string{},
		},
	}
}

func (p *GithubProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(p.sessStore, &p.oa2)(c)
}

func (p *GithubProvider) CallbackHandler(c echo.Context) error {
	return defaultCallbackHandler(p, &p.oa2, p.repo, p.fm, p.sessStore, p.config.RegisterUserIfNotFound)(c)
}

func (p *GithubProvider) FetchUserInfo(t *oauth2.Token) (UserInfo, error) {
	var ui githubUserInfo
	ui.p = p
	ui.t = t

	c := p.oa2.Client(context.Background(), t)

	resp, err := c.Get(githubProfileURL)
	if err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	var u struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Login   string `json:"login"`
		Picture string `json:"avatar_url"`
	}
	if err := json.ConfigFastest.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	ui.id = u.ID
	ui.name = u.Login
	ui.displayName = u.Name
	ui.profileImageURL = u.Picture

	return &ui, nil
}

func (p *GithubProvider) L() *zap.Logger {
	return p.logger
}
