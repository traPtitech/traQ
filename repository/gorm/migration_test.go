package gorm

import (
	"testing"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
)

func TestMigrationOpenIDUserInfoPermission(t *testing.T) {
	var target *gormigrate.Migration
	for _, m := range migration.Migrations() {
		if m.ID == "20261023" {
			target = m
		}
	}
	require.NotNil(t, target)
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing permission", true: "existing permission"}[existing], func(t *testing.T) {
			tx := repositories[ex3].db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			require.NoError(t, tx.Create(&model.UserRole{Name: "openid", Oauth2Scope: true, System: true}).Error)
			permission := &model.RolePermission{Role: "openid", Permission: "get_oidc_userinfo"}
			if existing {
				require.NoError(t, tx.Create(permission).Error)
			}
			require.NoError(t, target.Migrate(tx))
			var count int64
			require.NoError(t, tx.Model(permission).Where(permission).Count(&count).Error)
			require.EqualValues(t, 1, count)
		})
	}
}
