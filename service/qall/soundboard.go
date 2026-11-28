package qall

import (
	"context"
	"errors"
	"io"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// ErrInvalidSoundboardAudio アップロードされた音声が受け付けられない (形式が非対応・長すぎる・壊れている) ことを表します
var ErrInvalidSoundboardAudio = errors.New("invalid soundboard audio")

// MaxSoundboardDurationSeconds サウンドボードに登録できる音声の最大の長さ (秒)
const MaxSoundboardDurationSeconds = 30

type Soundboard interface {
	// SaveSoundboardItem サウンドボードアイテムを保存します
	// 成功した場合、nilを返します
	// 音声の形式が非対応・長すぎる・壊れている場合、ErrInvalidSoundboardAudioをラップしたエラーを返します
	// イベント通知機能が実装されている場合、アイテム作成時にQallSoundboardItemCreatedイベントが発行されます
	SaveSoundboardItem(ctx context.Context, soundID uuid.UUID, soundName string, contentType string, fileType model.FileType, src io.Reader, stampID *uuid.UUID, creatorID uuid.UUID) error
	// GetURL Pre-signed URLを取得します
	// 有効期限は5分です
	// 成功した場合、URLとnilを返します
	GetURL(soundID uuid.UUID) (string, error)
	// DeleteSoundboardItem サウンドボードアイテムを削除します
	// 成功した場合、nilを返します
	// イベント通知機能が実装されている場合、アイテム削除時にQallSoundboardItemDeletedイベントが発行されます
	DeleteSoundboardItem(ctx context.Context, soundID uuid.UUID) error
}
