package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// UserSettingsRepository ユーザセッティングレポジトリ
type UserSettingsRepository interface {
	// UpdateNotifyCitation メッセージ引用通知を設定します
	//
	// isEnableがtrueの場合、メッセージ引用通知を有効にします
	// isEnableがfalseの場合、メッセージ引用通知を無効にします
	// DBによるエラーを返すことがあります
	UpdateNotifyCitation(userID uuid.UUID, isEnable bool) error
	// GetNotifyCitation メッセージ引用通知の情報を取得します
	//
	// 返り値がtrueの場合、メッセージ引用通知が有効です
	// 返り値がfalseの場合、メッセージ引用通知が無効がエラーが発生しています
	// DBによるエラーを返すことがあります
	GetNotifyCitation(userID uuid.UUID) (bool, error)
	// GetUserSettings ユーザー設定を返します
	// DBによるエラーを返すことがあります
	GetUserSettings(userID uuid.UUID) (*model.UserSettings, error)
}
