package model

import "github.com/gofrs/uuid"

// UserSettings ユーザー設定の構造体
type UserSettings struct {
	UserID         uuid.UUID `gorm:"type:char(36);not null;primaryKey;" json:"id"`
	NotifyCitation bool      `gorm:"type:boolean" json:"notifyCitation"`

	User *User `gorm:"constraint:user_settings_user_id_users_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

// TableName UserSettings構造体のテーブル名
func (us *UserSettings) TableName() string {
	return "user_settings"
}

// IsNotifyCitationEnabled メッセージの引用通知が有効かどうかを返します
func (us *UserSettings) IsNotifyCitationEnabled() bool {
	return us.NotifyCitation
}
