package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"golang.org/x/oauth2"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"
)

const (
	OIDCProviderName          = "oidc"
	oidcAPIRequestErrorFormat = "oidc api request error: %w"
)

type OIDCProvider struct {
	config    OIDCProviderConfig
	repo      repository.Repository
	fm        file.Manager
	logger    *zap.Logger
	oa2       oauth2.Config
	sessStore session.Store
	oidc      *oidc.Provider
}

type OIDCProviderConfig struct {
	ClientID               string
	ClientSecret           string
	CallbackURL            string
	Issuer                 string
	Scopes                 []string
	RegisterUserIfNotFound bool
}

func (c OIDCProviderConfig) Valid() bool {
	return len(c.ClientSecret) > 0 && len(c.ClientID) > 0 && len(c.CallbackURL) > 0 && len(c.Issuer) > 0
}

type oidcUserInfo struct {
	p       *OIDCProvider
	t       *oauth2.Token
	sub     string
	name    string
	picture string
}

func (u *oidcUserInfo) GetProviderName() string {
	return OIDCProviderName
}

func (u *oidcUserInfo) GetID() string {
	return u.sub
}

func (u *oidcUserInfo) GetRawName() string {
	return u.name
}

func (u *oidcUserInfo) GetName() string {
	s := strings.ReplaceAll(u.name, " ", "")
	regex := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	s = regex.ReplaceAllLiteralString(s, "_")
	if us := utf8string.NewString(s); us.RuneCount() > 32 {
		s = us.Slice(0, 32)
	}
	return s
}

func (u *oidcUserInfo) GetDisplayName() string {
	if s := utf8string.NewString(u.name); s.RuneCount() > 32 {
		return s.Slice(0, 32)
	}
	return u.name
}

func (u *oidcUserInfo) GetProfileImage() ([]byte, error) {
	if len(u.picture) == 0 {
		return nil, nil
	}
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(u.picture)
	if err != nil {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *oidcUserInfo) IsLoginAllowedUser() bool {
	return true // TODO
}

func NewOIDCProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config OIDCProviderConfig) (*OIDCProvider, error) {
	p, err := oidc.NewProvider(context.Background(), config.Issuer)
	if err != nil {
		return nil, err
	}

	return &OIDCProvider{
		repo:      repo,
		fm:        fm,
		config:    config,
		logger:    logger,
		sessStore: sessStore,
		oidc:      p,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.CallbackURL,
			Endpoint:     p.Endpoint(),
			Scopes:       append([]string{oidc.ScopeOpenID}, config.Scopes...),
		},
	}, nil
}

func (p *OIDCProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(p.sessStore, &p.oa2)(c)
}

func (p *OIDCProvider) CallbackHandler(c echo.Context) error {
	return defaultCallbackHandler(p, &p.oa2, p.repo, p.fm, p.sessStore, p.config.RegisterUserIfNotFound)(c)
}

func (p *OIDCProvider) FetchUserInfo(t *oauth2.Token) (UserInfo, error) {
	var ui oidcUserInfo
	ui.p = p
	ui.t = t

	verifier := p.oidc.Verifier(&oidc.Config{ClientID: p.config.ClientID})

	rawIDToken, ok := t.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, errors.New("missing id_token"))
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, err)
	}

	// Extract custom claims
	var claims struct {
		Sub     string              `json:"sub"`
		Name    string              `json:"name"`
		Picture optional.Of[string] `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf(oidcAPIRequestErrorFormat, errors.New("malformed id_token"))
	}
	ui.sub = claims.Sub
	ui.name = claims.Name
	ui.picture = claims.Picture.V

	return &ui, nil
}

func (p *OIDCProvider) L() *zap.Logger {
	return p.logger
}
