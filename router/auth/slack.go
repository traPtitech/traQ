package auth

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	json "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	slackOAuth2 "golang.org/x/oauth2/slack"
)

const (
	SlackProviderName             = "slack"
	slackAPIRequestErrorFormat    = "slack api request error: %w"
	slackAPIUsersInfoEndpoint     = "https://slack.com/api/users.info"
)

type SlackProvider struct {
	config    SlackProviderConfig
	repo      repository.Repository
	fm        file.Manager
	logger    *zap.Logger
	sessStore session.Store
	oa2       oauth2.Config
}

type SlackProviderConfig struct {
	ClientID               string
	ClientSecret           string
	CallbackURL            string
	RegisterUserIfNotFound bool
}

func (c SlackProviderConfig) Valid() bool {
	return len(c.ClientSecret) > 0 && len(c.ClientID) > 0 && len(c.CallbackURL) > 0
}

type slackUserInfo struct {
	p               *SlackProvider
	t               *oauth2.Token
	id              string
	displayName     string
	profileImageURL string
	teamID          string
}

func (u *slackUserInfo) GetProviderName() string {
	return SlackProviderName
}

func (u *slackUserInfo) GetID() string {
	return u.id
}

func (u *slackUserInfo) GetRawName() string {
	return u.displayName
}

func (u *slackUserInfo) GetName() string {
	return u.displayName
}

func (u *slackUserInfo) GetDisplayName() string {
	return u.displayName
}

func (u *slackUserInfo) GetProfileImage() ([]byte, error) {
	if len(u.profileImageURL) == 0 {
		return nil, nil
	}
	c := u.p.oa2.Client(context.Background(), u.t)
	resp, err := c.Get(u.profileImageURL)
	if err != nil {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, err)
	}
	return b, nil
}

func (u *slackUserInfo) IsLoginAllowedUser() bool {
	return true
}

func NewSlackProvider(repo repository.Repository, fm file.Manager, logger *zap.Logger, sessStore session.Store, config SlackProviderConfig) *SlackProvider {
	return &SlackProvider{
		config:    config,
		repo:      repo,
		fm:        fm,
		logger:    logger,
		sessStore: sessStore,
		oa2: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.CallbackURL,
			Endpoint:     slackOAuth2.Endpoint,
			Scopes:       []string{"users:read"},
		},
	}
}

func (p *SlackProvider) LoginHandler(c echo.Context) error {
	return defaultLoginHandler(p.sessStore, &p.oa2)(c)
}

func (p *SlackProvider) CallbackHandler(c echo.Context) error {
	return defaultCallbackHandler(p, &p.oa2, p.repo, p.fm, p.sessStore, p.config.RegisterUserIfNotFound)(c)
}

func (p *SlackProvider) FetchUserInfo(t *oauth2.Token) (UserInfo, error) {
	var ui slackUserInfo
	ui.p = p
	ui.t = t
	ui.id = t.Extra("user_id").(string)

	c := p.oa2.Client(context.Background(), t)
	resp, err := c.Get(slackAPIUsersInfoEndpoint + "?user=" + ui.id)
	if err != nil {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, fmt.Errorf("invalid status code: %d", resp.StatusCode))
	}

	var data struct {
		OK   bool `json:"ok"`
		User struct {
			TeamID  string `json:"team_id"`
			Profile struct {
				DisplayName string `json:"display_name"`
				Image512    string `json:"image_512"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.ConfigFastest.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf(slackAPIRequestErrorFormat, err)
	}
	log.Println(data)
	ui.displayName = data.User.Profile.DisplayName
	ui.profileImageURL = data.User.Profile.Image512
	ui.teamID = data.User.TeamID
	return &ui, nil
}

func (p *SlackProvider) L() *zap.Logger {
	return p.logger
}
