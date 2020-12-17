//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// PinRepository ピンリポジトリ
type PinRepository interface {
	// PinMessage 指定したユーザーによって指定したメッセージをピン留めします
	PinMessage(messageID, userID uuid.UUID) (*model.Pin, error)
	// UnpinMessage 指定したユーザーによって指定したピン留めを削除します
	UnpinMessage(messageID uuid.UUID) (*model.Pin, error)
	// GetPinnedMessageByChannelID 指定したチャンネルのピン留めを全て取得します
	GetPinnedMessageByChannelID(channelID uuid.UUID) ([]*model.Pin, error)
}
