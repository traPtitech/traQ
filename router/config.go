package router

import (
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/router/oauth2"
	v3 "github.com/traPtitech/traQ/router/v3"
)

// Config APIサーバー設定
type Config struct {
	// Origin サーバーオリジン (e.g. https://q.trap.jp)
	Origin string
	// 開発モードかどうか
	Development bool
	// Version サーバーバージョン
	Version string
	// Revision サーバーリビジョン
	Revision string
	// AccessLogging アクセスログを記録するかどうか
	AccessLogging bool
	// Gzipped レスポンスをGzip圧縮するかどうか
	Gzipped bool
	// AllowSignUp ユーザーが自分自身で登録できるかどうか
	AllowSignUp bool
	// AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	// IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool
	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string
	// LiveKitHost LiveKitホスト
	LiveKitHost string
	// LiveKitApiKey LiveKit APIキー
	LiveKitApiKey string
	// LiveKitApiSecret LiveKit APIシークレット
	LiveKitApiSecret string
	// ExternalAuth 外部認証設定
	ExternalAuth ExternalAuthConfig
}

// ExternalAuthConfig 外部認証設定
type ExternalAuthConfig struct {
	// GitHub GitHub OAuth2
	GitHub auth.GithubProviderConfig
	// Google Google OAuth2
	Google auth.GoogleProviderConfig
	// TraQ TraQ OAuth2
	TraQ auth.TraQProviderConfig
	// OIDC OpenID Connect
	OIDC auth.OIDCProviderConfig
	// Slack Slack OAuth2
	Slack auth.SlackProviderConfig
}

func (c ExternalAuthConfig) ValidProviders() map[string]bool {
	res := make(map[string]bool)
	if c.GitHub.Valid() {
		res[auth.GithubProviderName] = true
	}
	if c.Google.Valid() {
		res[auth.GoogleProviderName] = true
	}
	if c.TraQ.Valid() {
		res[auth.TraQProviderName] = true
	}
	if c.OIDC.Valid() {
		res[auth.OIDCProviderName] = true
	}
	if c.Slack.Valid() {
		res[auth.SlackProviderName] = true
	}
	return res
}

func provideOAuth2Config(c *Config) oauth2.Config {
	return oauth2.Config{
		Origin:           c.Origin,
		AccessTokenExp:   c.AccessTokenExp,
		IsRefreshEnabled: c.IsRefreshEnabled,
	}
}

func provideV3Config(c *Config) v3.Config {
	return v3.Config{
		Version:                         c.Version,
		Revision:                        c.Revision,
		SkyWaySecretKey:                 c.SkyWaySecretKey,
		LiveKitHost:                     c.LiveKitHost,
		LiveKitApiKey:                   c.LiveKitApiKey,
		LiveKitApiSecret:                c.LiveKitApiSecret,
		AllowSignUp:                     c.AllowSignUp,
		EnabledExternalAccountProviders: c.ExternalAuth.ValidProviders(),
	}
}
