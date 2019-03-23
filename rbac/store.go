package rbac

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"time"
)

// Store : RBACのルール永続化用ストアインターフェイス
type Store interface {
	OverrideStore
}

// OverrideStore : RBACのオーバーライドルールの永続化用ストアインターフェイス
type OverrideStore interface {
	GetAllOverrides() ([]OverrideData, error)
	SaveOverride(userID uuid.UUID, p gorbac.Permission, validity bool) error
	DeleteOverride(userID uuid.UUID, p gorbac.Permission) error
}

// GormStore rbac.OverrideStoreインターフェイスの実装
type GormStore struct {
	db *gorm.DB
}

// NewDefaultStore RBACのルール永続化用ストアを生成します
func NewDefaultStore(db *gorm.DB) (Store, error) {
	s := &GormStore{
		db: db,
	}
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(&Override{}).Error; err != nil {
		return nil, fmt.Errorf("failed to sync Table schema: %v", err)
	}
	return s, nil
}

// Override rbac.OverrideDataインターフェイスの実装
type Override struct {
	UserID     uuid.UUID `gorm:"type:char(36);primary_key"`
	Permission string    `gorm:"type:varchar(50);primary_key"`
	Validity   bool
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

// TableName RBACのオーバライドルールのテーブル名
func (*Override) TableName() string {
	return "rbac_overrides"
}

// GetUserID ユーザーIDを取得
func (o *Override) GetUserID() uuid.UUID {
	return o.UserID
}

// GetPermission パーミッションを取得
func (o *Override) GetPermission() gorbac.Permission {
	return permission.GetPermission(o.Permission)
}

// GetValidity 有効性を取得
func (o *Override) GetValidity() bool {
	return o.Validity
}

// GetAllOverrides オーバライドルールを全て取得します
func (s *GormStore) GetAllOverrides() ([]OverrideData, error) {
	var overrides []*Override
	if err := s.db.Find(&overrides).Error; err != nil {
		return nil, err
	}

	res := make([]OverrideData, len(overrides))
	for i, v := range overrides {
		res[i] = v
	}
	return res, nil
}

// SaveOverride オーバライドルール保存します
func (s *GormStore) SaveOverride(userID uuid.UUID, p gorbac.Permission, validity bool) error {
	return s.db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE validity = VALUES(validity), updated_at = now()").
		Create(&Override{UserID: userID, Permission: p.ID(), Validity: validity}).
		Error
}

// DeleteOverride オーバライドルールを削除します
func (s *GormStore) DeleteOverride(userID uuid.UUID, p gorbac.Permission) error {
	return s.db.Where(&Override{UserID: userID, Permission: p.ID()}).Delete(Override{}).Error
}
