package model

import (
	"fmt"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"time"
)

// RBACOverride rbac.OverrideDataインターフェイスの実装
type RBACOverride struct {
	UserID     string `gorm:"type:char(36);primary_key"`
	Permission string `gorm:"type:varchar(50);primary_key"`
	Validity   bool
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

// RBACOverrideStore rbac.OverrideStoreインターフェイスの実装
type RBACOverrideStore struct{}

// TableName RBACのオーバライドルールのテーブル名
func (*RBACOverride) TableName() string {
	return "rbac_overrides"
}

// GetUserID ユーザーIDを取得
func (o *RBACOverride) GetUserID() uuid.UUID {
	return uuid.Must(uuid.FromString(o.UserID))
}

// GetPermission パーミッションを取得
func (o *RBACOverride) GetPermission() gorbac.Permission {
	return permission.GetPermission(o.Permission)
}

// GetValidity 有効性を取得
func (o *RBACOverride) GetValidity() bool {
	return o.Validity
}

// GetAllOverrides オーバライドルールを全て取得します
func (*RBACOverrideStore) GetAllOverrides() ([]rbac.OverrideData, error) {
	var overrides []*RBACOverride
	if err := db.Find(&overrides).Error; err != nil {
		return nil, err
	}

	res := make([]rbac.OverrideData, len(overrides))
	for i, v := range overrides {
		res[i] = v
	}
	return res, nil
}

// SaveOverride オーバライドルール保存します
func (*RBACOverrideStore) SaveOverride(userID uuid.UUID, p gorbac.Permission, validity bool) error {
	return db.
		Set("gorm:insert_option", fmt.Sprintf("ON DUPLICATE KEY UPDATE validity = %v, updated_at = now()", validity)).
		Create(RBACOverride{UserID: userID.String(), Permission: p.ID(), Validity: validity}).
		Error
}

// DeleteOverride オーバライドルールを削除します
func (*RBACOverrideStore) DeleteOverride(userID uuid.UUID, p gorbac.Permission) error {
	return db.Where(RBACOverride{UserID: userID.String(), Permission: p.ID()}).Delete(RBACOverride{}).Error
}
