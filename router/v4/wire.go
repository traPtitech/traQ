//go:build wireinject
// +build wireinject

package v4

import (
	"github.com/google/wire"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/v4/messages"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	mutil "github.com/traPtitech/traQ/utils/message"
)

// Handlersの構築（認証なしバージョン）
func ProvideHandlers(
	messageService *messages.Service,
) *Handlers {
	return &Handlers{
		MessageService: messageService,
	}
}

// wireセット
var V4Set = wire.NewSet(
	messages.NewService,
	ProvideHandlers,
)

// wire初期化関数
func InitializeHandlers(
	repo repository.Repository,
	channelManager channel.Manager,
	messageManager message.Manager,
	replacer *mutil.Replacer,
) *Handlers {
	wire.Build(V4Set)
	return nil
}
