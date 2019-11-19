package router

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sse"
	"go.uber.org/zap"
)

// Config APIサーバー設定
type Config struct {
	// Version サーバーバージョン
	Version string
	// Revision サーバーリビジョン
	Revision string
	// AccessLogging アクセスログを記録するかどうか
	AccessLogging bool
	// Gzipped レスポンスをGzip圧縮するかどうか
	Gzipped bool
	// ImageMagickPath ImageMagickの実行パス
	ImageMagickPath string
	// AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	// IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool
	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string
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
	Realtime *realtime.Service
	// RootLogger ルートロガー
	RootLogger *zap.Logger
}
