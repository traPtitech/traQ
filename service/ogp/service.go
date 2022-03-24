package ogp

import (
	"net/url"
	"time"

	"github.com/traPtitech/traQ/model"
)

// Service OGPサービス
type Service interface {
	// Shutdown OGPサービスを停止します
	Shutdown() error

	// GetMeta 指定したURLのメタタグをパースした結果を返します。
	//
	// 成功した場合、*model.Ogp、expiresIn、nil を返します。
	// URLに対応する情報が存在しない場合、nil、expiresIn、nilを返します。
	GetMeta(url *url.URL) (ogp *model.Ogp, expiresIn time.Duration, err error)
}
