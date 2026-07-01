package repository

import (
	"context"
	"time"

	"github.com/traPtitech/traQ/model"
)

type OgpCacheRepository interface {
	// CreateOgpCache OGPキャッシュを作成します
	//
	// contentがnilの場合、Validをfalseとしたネガティブキャッシュを作成します。
	//
	// 成功した場合、作成されたOGPキャッシュとnilを返します。
	// DBによるエラーを返すことがあります。
	CreateOgpCache(ctx context.Context, url string, content *model.Ogp, cacheFor time.Duration) (c *model.OgpCache, err error)

	// GetOgpCache 指定したURLのOGPキャッシュを取得します
	//
	// 成功した場合、取得したOGPキャッシュとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetOgpCache(ctx context.Context, url string) (c *model.OgpCache, err error)

	// DeleteOgpCache 指定したURLのOGPキャッシュを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	DeleteOgpCache(ctx context.Context, url string) error

	// DeleteStaleOgpCache 保存期間が経過したOGPキャッシュを削除します
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	DeleteStaleOgpCache(ctx context.Context) error
}
