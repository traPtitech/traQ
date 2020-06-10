package bot

import "context"

// Service BOTサービス
type Service interface {
	// Start イベントの発送を開始します
	Start()
	// Shutdown BOTサービスをシャットダウンします
	Shutdown(ctx context.Context) error
}
