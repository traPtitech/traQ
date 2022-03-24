package bot

import "context"

// Service BOTサービス
type Service interface {
	// Shutdown BOTサービスをシャットダウンします
	Shutdown(ctx context.Context) error
}
