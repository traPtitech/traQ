package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
)

const (
	GithubProviderName          = "github"
	githubProfileURL            = "https://api.github.com/user"
	githubUserOrgsURL           = "https://api.github.com/user/orgs"
	githubScopeReadOrg          = "read:org"
	githubAPIRequestErrorFormat = "github api request error: %w"
)

type GithubProvider struct {
	config               GithubProviderConfig
	repo                 repository.Repository
	fm                   file.Manager
	logger               *zap.Logger
	sessStore            session.Store
	oa2                  oauth2.Config
	allowedOrganizations []string
}

type GithubProviderConfig struct {
	ClientID               string
	ClientSecret           string
	AllowedOrganizations   []string
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
	organizations   []string
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
	if s := utf8string.NewString(u.displayName); s.RuneCount() > 32 {
		return s.Slice(0, 32)
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *githubUserInfo) IsLoginAllowedUser() bool {
	if len(u.p.allowedOrganizations) > 0 {
		for _, allowedOrg := range u.p.allowedOrganizations {
			for _, org := range u.organizations {
				if org == allowedOrg {
					return true
				}
			}
		}

		return false
	}

	return true
}

func (u *githubUserInfo) fetchUserInfo() error {
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(githubProfileURL)
	if err != nil {
		return fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(githubAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	var user struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Login   string `json:"login"`
		Picture string `json:"avatar_url"`
	}
	if err := json.ConfigFastest.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	u.id = user.ID
	u.name = user.Login
	u.displayName = user.Name
	u.profileImageURL = user.Picture
	return nil
}

func (u *githubUserInfo) fetchOrganizations() error {
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(githubUserOrgsURL)
	if err != nil {
		return fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(githubAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	var orgs []struct {
		Login string `json:"login"`
	}
	if err := json.ConfigFastest.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return fmt.Errorf(githubAPIRequestErrorFormat, err)
	}
	u.organizations = make([]string, len(orgs))
	for i, org := range orgs {
		u.organizations[i] = org.Login
	}
	return nil
}

func NewGithubProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config GithubProviderConfig) *GithubProvider {
	scopes := make([]string, 0)
	if len(config.AllowedOrganizations) > 0 {
		scopes = append(scopes, githubScopeReadOrg)
	}
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
			Scopes:       scopes,
		},
		allowedOrganizations: config.AllowedOrganizations,
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

	err := ui.fetchUserInfo()
	if err != nil {
		return nil, err
	}
	if len(p.allowedOrganizations) > 0 {
		err = ui.fetchOrganizations()
		if err != nil {
			return nil, err
		}
	}

	return &ui, nil
}

func (p *GithubProvider) L() *zap.Logger {
	return p.logger
}
