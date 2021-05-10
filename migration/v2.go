package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/service/rbac/role"
)

// v2 RBAC周りのリフォーム
func v2() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "2",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v2UserRole{}, &v2RoleInheritance{}, &v2RolePermission{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"user_role_inheritances", "user_role_inheritances_role_user_roles_name_foreign", "role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"user_role_inheritances", "user_role_inheritances_sub_role_user_roles_name_foreign", "sub_role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"user_role_permissions", "user_role_permissions_role_user_roles_name_foreign", "role", "user_roles(name)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			for _, v := range role.SystemRoleModels() {
				if err := db.Create(v).Error; err != nil {
					return err
				}
				for _, v := range v.Permissions {
					if err := db.Create(v).Error; err != nil {
						return err
					}
				}
			}

			return db.Migrator().DropTable(&v2Override{})
		},
	}
}

type v2UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primaryKey"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`
}

func (*v2UserRole) TableName() string {
	return "user_roles"
}

type v2RoleInheritance struct {
	Role    string `gorm:"type:varchar(30);not null;primaryKey"`
	SubRole string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v2RoleInheritance) TableName() string {
	return "user_role_inheritances"
}

type v2RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v2RolePermission) TableName() string {
	return "user_role_permissions"
}

type v2Override struct {
	UserID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	Permission string    `gorm:"type:varchar(50);primaryKey"`
	Validity   bool
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

func (*v2Override) TableName() string {
	return "rbac_overrides"
}
