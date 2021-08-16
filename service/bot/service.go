package bot

import "context"

// Service BOTサービス
type Service interface {
	// Start BOTサービスを開始します
	Start()
	// Shutdown BOTサービスをシャットダウンします
	Shutdown(ctx context.Context) error
}
