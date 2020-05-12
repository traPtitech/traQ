package router

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/auth"
	"github.com/traPtitech/traQ/service"
	imaging2 "github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/ws"
	"go.uber.org/zap"
)

// Config APIサーバー設定
type Config struct {
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
	// AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	// IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool
	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string
	// ExternalAuth 外部認証設定
	ExternalAuth ExternalAuthConfig
	// Hub イベントハブ
	Hub *hub.Hub
	// Repository リポジトリ
	Repository repository.Repository
	// RBAC アクセスコントローラー
	RBAC rbac.RBAC
	// WS WebSocketストリーマー
	WS *ws.Streamer
	// SSE SSEストリーマー
	SSE *sse.Streamer
	// Realtime リアルタイムサービス
	Realtime *service.Services
	// RootLogger ルートロガー
	RootLogger *zap.Logger
	// Imaging 画像処理機
	Imaging imaging2.Processor
}

// ExternalAuth 外部認証設定
type ExternalAuthConfig struct {
	// GitHub GitHub OAuth2
	GitHub auth.GithubProviderConfig
	// Google Google OAuth2
	Google auth.GoogleProviderConfig
	// TraQ TraQ OAuth2
	TraQ auth.TraQProviderConfig
	// OIDC OpenID Connect
	OIDC auth.OIDCProviderConfig
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
	return res
}
