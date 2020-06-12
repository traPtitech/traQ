//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package channel

import (
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// Tree 公開チャンネルのチャンネル階層木
type Tree interface {
	// GetModel 指定したチャンネルの*model.Channelを取得する
	GetModel(id uuid.UUID) (*model.Channel, error)
	// GetChildrenIDs 子チャンネルのIDの配列を取得する
	GetChildrenIDs(id uuid.UUID) []uuid.UUID
	// GetDescendantIDs 子孫チャンネルのIDの配列を取得する
	GetDescendantIDs(id uuid.UUID) []uuid.UUID
	// GetAscendantIDs 祖先チャンネルのIDの配列を取得する
	GetAscendantIDs(id uuid.UUID) []uuid.UUID
	// GetChannelDepth 指定したチャンネル木の深さを取得する
	GetChannelDepth(id uuid.UUID) int
	// IsChildPresent 指定したnameのチャンネルが指定したチャンネルの子に存在するか
	IsChildPresent(name string, parent uuid.UUID) bool
	// GetChannelPath 指定したチャンネルのパスを取得する
	GetChannelPath(id uuid.UUID) string
	// IsChannelPresent 指定したIDのチャンネルが存在するかどうかを取得する
	IsChannelPresent(id uuid.UUID) bool
	// GetChannelIDFromPath チャンネルパスからチャンネルIDを取得する
	GetChannelIDFromPath(path string) uuid.UUID
	// IsForceChannel 指定したチャンネルが強制通知チャンネルかどうか
	IsForceChannel(id uuid.UUID) bool
	// IsArchivedChannel 指定したチャンネルがアーカイブされているかどうか
	IsArchivedChannel(id uuid.UUID) bool
	json.Marshaler
}
