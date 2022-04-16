package ogp

import (
	"net/url"
	"time"

	"github.com/traPtitech/traQ/model"
)

const DefaultCacheDuration = time.Hour * 24 * 7

// Service OGPサービス
type Service interface {
	// Shutdown OGPサービスを停止します
	Shutdown() error

	// GetMeta 指定したURLのメタタグをパースした結果を返します。
	//
	// 成功した場合、*model.Ogp、expiresAt、nil を返します。
	// URLに対応する情報が存在しない場合、nil、expiresAt、nilを返します。
	// 情報が存在する場合としない場合両方において expiresAt までキャッシュが可能です。
	//
	// 内部エラーが発生した場合、nil, 0, err を返します。
	GetMeta(url *url.URL) (ogp *model.Ogp, expiresAt time.Time, err error)
}
