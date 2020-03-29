package auth

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	githubProviderName          = "github"
	githubProfileURL            = "https://api.github.com/user"
	githubAPIRequestErrorFormat = "github api request error: %w"
)

type GithubProvider struct {
	config GithubProviderConfig
	repo   repository.Repository
	logger *zap.Logger
	oa2    oauth2.Config
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
	return githubProviderName
}

func (u *githubUserInfo) GetID() string {
	return strconv.Itoa(u.id)
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

func NewGithubProvider(repo repository.Repository, logger *zap.Logger, config GithubProviderConfig) *GithubProvider {
	return &GithubProvider{
		repo:   repo,
		config: config,
		logger: logger,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       []string{},
		},
	}
}

func (p *GithubProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(&p.oa2)(c)
}

func (p *GithubProvider) CallbackHandler(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	if len(code) == 0 || len(state) == 0 {
		return herror.BadRequest("missing code or state")
	}

	cookie, err := c.Cookie(cookieName)
	if err != nil {
		return herror.BadRequest("missing cookie")
	}
	if cookie.Value != state {
		return herror.BadRequest("invalid state")
	}

	t, err := p.oa2.Exchange(context.Background(), code)
	if err != nil {
		return herror.BadRequest("token exchange failed")
	}

	tu, err := p.FetchUserInfo(t)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if !tu.IsLoginAllowedUser() {
		return c.String(http.StatusForbidden, "You are not permitted to access traQ")
	}

	user, err := p.repo.GetUserByExternalID(tu.GetProviderName(), tu.GetID(), false)
	if err != nil {
		if err != repository.ErrNotFound {
			return herror.InternalServerError(err)
		}

		if !p.config.RegisterUserIfNotFound {
			return herror.Unauthorized("You are not a member of traQ")
		}

		args := repository.CreateUserArgs{
			Name:        tu.GetName(),
			DisplayName: tu.GetDisplayName(),
			Role:        role.User,
			ExternalLogin: &model.ExternalProviderUser{
				ProviderName: tu.GetProviderName(),
				ExternalID:   tu.GetID(),
			},
		}

		if b, err := tu.GetProfileImage(); err == nil && b != nil {
			fid, err := processProfileIcon(p.repo, b)
			if err == nil {
				args.IconFileID = uuid.NullUUID{Valid: true, UUID: fid}
			}
		}

		user, err = p.repo.CreateUser(args)
		if err != nil {
			if err == repository.ErrAlreadyExists {
				return herror.Conflict("name conflicts") // TODO 名前被りをどうするか
			}
			return herror.InternalServerError(err)
		}
		p.logger.Info("New user was created by external auth (github)",
			zap.Stringer("id", user.GetID()),
			zap.String("name", user.GetName()),
			zap.String("githubId", tu.GetID()),
			zap.String("githubLogin", tu.GetName()))
	}

	// ユーザーのアカウント状態の確認
	if !user.IsActive() {
		return herror.Forbidden("this account is currently suspended")
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if err := sess.SetUser(user.GetID()); err != nil {
		return herror.InternalServerError(err)
	}
	p.logger.Info("User was logged in by external auth (github)",
		zap.Stringer("id", user.GetID()),
		zap.String("name", user.GetName()),
		zap.String("githubId", tu.GetID()),
		zap.String("githubLogin", tu.GetName()))

	return c.Redirect(http.StatusFound, "/")
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
