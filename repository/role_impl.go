package repository

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"regexp"
)

var roleNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,30}$`)

// GetAllRoles implements UserRoleRepository interface.
func (repo *GormRepository) GetAllRoles() ([]*model.UserRole, error) {
	result := make([]*model.UserRole, 0)
	err := repo.db.
		Preload("Inheritances").
		Preload("Permissions").
		Find(&result).
		Error
	return result, err
}

// GetRole implements UserRoleRepository interface.
func (repo *GormRepository) GetRole(role string) (*model.UserRole, error) {
	if len(role) == 0 {
		return nil, ErrNotFound
	}
	var r model.UserRole
	if err := repo.db.
		Preload("Inheritances").
		Preload("Permissions").
		First(&r, &model.UserRole{Name: role}).
		Error; err != nil {
		return nil, convertError(err)
	}
	return &r, nil
}

// CreateRole implements UserRoleRepository interface.
func (repo *GormRepository) CreateRole(name string) error {
	if !roleNameRegex.MatchString(name) {
		return ArgError("name", "Name must be 1-30 characters of a-zA-Z0-9_")
	}

	// 名前重複チェック
	if exists, err := dbExists(repo.db, &model.UserRole{Name: name}); err != nil {
		return err
	} else if exists {
		return ErrAlreadyExists
	}

	return repo.db.Create(&model.UserRole{Name: name}).Error
}

// UpdateRole implements UserRoleRepository interface.
func (repo *GormRepository) UpdateRole(role string, args UpdateRoleArgs) error {
	if len(role) == 0 {
		return ErrNotFound
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var r model.UserRole
		if err := tx.First(&r, &model.UserRole{Name: role}).Error; err != nil {
			return convertError(err)
		}

		if args.OAuth2Scope.Valid {
			if err := tx.Model(&r).Update("oauth2_scope", args.OAuth2Scope.Bool).Error; err != nil {
				return err
			}
		}

		if args.Permissions != nil {
			if err := tx.Delete(&model.RolePermission{Role: role}).Error; err != nil {
				return err
			}
			for _, v := range args.Permissions {
				if err := tx.Create(&model.RolePermission{Role: role, Permission: v}).Error; err != nil {
					return err
				}
			}
		}

		if args.Inheritances != nil {
			if err := tx.Delete(&model.RoleInheritance{Role: role}).Error; err != nil {
				return err
			}
			for _, v := range args.Inheritances {
				if err := tx.Create(&model.RoleInheritance{Role: role, SubRole: v}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
