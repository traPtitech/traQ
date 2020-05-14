package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/service/rbac/role"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v2 RBAC周りのリフォーム
func v2() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "2",
		Migrate: func(db *gorm.DB) error {
			if err := db.AutoMigrate(&v2UserRole{}, &v2RoleInheritance{}, &v2RolePermission{}).Error; err != nil {
				return err
			}

			foreignKeys := [][5]string{
				{"user_role_inheritances", "role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"user_role_inheritances", "sub_role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"user_role_permissions", "role", "user_roles(name)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
					return err
				}
			}

			for _, v := range role.SystemRoles() {
				if err := db.Create(v).Error; err != nil {
					return err
				}
				for _, v := range v.Permissions {
					if err := db.Create(v).Error; err != nil {
						return err
					}
				}
			}

			return db.DropTableIfExists(&v2Override{}).Error
		},
	}
}

type v2UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primary_key"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`
}

func (*v2UserRole) TableName() string {
	return "user_roles"
}

type v2RoleInheritance struct {
	Role    string `gorm:"type:varchar(30);not null;primary_key"`
	SubRole string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v2RoleInheritance) TableName() string {
	return "user_role_inheritances"
}

type v2RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v2RolePermission) TableName() string {
	return "user_role_permissions"
}

type v2Override struct {
	UserID     uuid.UUID `gorm:"type:char(36);primary_key"`
	Permission string    `gorm:"type:varchar(50);primary_key"`
	Validity   bool
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

func (*v2Override) TableName() string {
	return "rbac_overrides"
}
