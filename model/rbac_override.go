package model

import (
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"time"
)

// RBACOverride rbac.OverrideDataインターフェイスの実装
type RBACOverride struct {
	UserID     string    `xorm:"char(36) not null unique(user_permission)"`
	Permission string    `xorm:"varchar(50) not null unique(user_permission)"`
	Validity   bool      `xorm:"bool not null"`
	CreatedAt  time.Time `xorm:"created"`
}

// RBACOverrideStore : rbac.OverrideStoreインターフェイスの実装
type RBACOverrideStore struct{}

// TableName : RBACのオーバライドルールのテーブル名
func (*RBACOverride) TableName() string {
	return "rbac_overrides"
}

// GetUserID : ユーザーIDを取得
func (o *RBACOverride) GetUserID() uuid.UUID {
	return uuid.FromStringOrNil(o.UserID)
}

// GetPermission : パーミッションを取得
func (o *RBACOverride) GetPermission() gorbac.Permission {
	return permission.GetPermission(o.Permission)
}

// GetValidity : 有効性を取得
func (o *RBACOverride) GetValidity() bool {
	return o.Validity
}

// GetAllOverrides : オーバライドルールを全て取得します
func (*RBACOverrideStore) GetAllOverrides() ([]rbac.OverrideData, error) {
	var overrides []*RBACOverride
	if err := db.Find(&overrides); err != nil {
		return nil, err
	}

	res := make([]rbac.OverrideData, len(overrides))
	for i, v := range overrides {
		res[i] = v
	}
	return res, nil
}

// SaveOverride : オーバライドルール保存します
func (*RBACOverrideStore) SaveOverride(userID uuid.UUID, p gorbac.Permission, validity bool) error {
	var or RBACOverride
	ok, err := db.Where("user_id = ? AND permission = ?", userID, p.ID()).Get(&or)
	if err != nil {
		return err
	}

	if ok {
		if _, err := db.UseBool().Where("user_id = ? AND permission = ?", userID, p.ID()).Update(&RBACOverride{Validity: validity}); err != nil {
			return err
		}
	} else {
		or.UserID = userID.String()
		or.Permission = p.ID()
		or.Validity = validity
		if _, err := db.UseBool().MustCols().InsertOne(&or); err != nil {
			return err
		}
	}

	return nil
}

// DeleteOverride : オーバライドルールを削除します
func (*RBACOverrideStore) DeleteOverride(userID uuid.UUID, p gorbac.Permission) (err error) {
	_, err = db.Delete(&RBACOverride{
		UserID:     userID.String(),
		Permission: p.ID(),
	})
	return
}
