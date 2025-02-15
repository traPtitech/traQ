package qall

import (
	"io"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

type Soundboard interface {
	// SaveSoundboardItem サウンドボードアイテムを保存します
	// 成功した場合、nilを返します
	SaveSoundboardItem(soundID uuid.UUID, soundName string, contentType string, fileType model.FileType, src io.Reader, stampID *uuid.UUID, creatorID uuid.UUID) error
	// Get Pre-signed URLを取得します
	// 有効期限は5分です
	// 成功した場合、URLとnilを返します
	GetURL(soundID uuid.UUID) (string, error)
	// DeleteSoundboardItem サウンドボードアイテムを削除します
	// 成功した場合、nilを返します
	DeleteSoundboardItem(soundID uuid.UUID) error
}
