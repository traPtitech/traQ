package qall

import (
	"io"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

type Soundboard interface {
	// SaveSoundboardItem サウンドボードアイテムを保存します
	// 成功した場合、nilを返します
	// イベント通知機能が実装されている場合、アイテム作成時にQallSoundboardItemCreatedイベントが発行されます
	SaveSoundboardItem(soundID uuid.UUID, soundName string, contentType string, fileType model.FileType, src io.Reader, stampID *uuid.UUID, creatorID uuid.UUID) error
	// GetURL Pre-signed URLを取得します
	// 有効期限は5分です
	// 成功した場合、URLとnilを返します
	GetURL(soundID uuid.UUID) (string, error)
	// DeleteSoundboardItem サウンドボードアイテムを削除します
	// 成功した場合、nilを返します
	// イベント通知機能が実装されている場合、アイテム削除時にQallSoundboardItemDeletedイベントが発行されます
	DeleteSoundboardItem(soundID uuid.UUID) error
}
