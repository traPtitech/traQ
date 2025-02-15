package qall

import "io"

type Soundboard interface {
	// SaveSoundboardItem サウンドボードアイテムを保存します
	// 成功した場合、nilを返します
	SaveSoundboardItem(soundID string, src io.Reader) error
	// Get Pre-signed URLを取得します
	// 有効期限は5分です
	// 成功した場合、URLとnilを返します
	GetURL(soundID string) (string, error)
	// DeleteSoundboardItem サウンドボードアイテムを削除します
	// 成功した場合、nilを返します
	DeleteSoundboardItem(soundID string) error
}
