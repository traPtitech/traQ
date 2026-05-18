package gorm

import (
	"context"

	"github.com/traPtitech/traQ/model"
)

// CreateUserRoles implements UserRoleRepository interface.
func (repo *Repository) CreateUserRoles(ctx context.Context, roles ...*model.UserRole) error {
	return repo.db.WithContext(ctx).Create(roles).Error
}

// GetAllUserRoles implements UserRoleRepository interface.
func (repo *Repository) GetAllUserRoles(ctx context.Context) ([]*model.UserRole, error) {
	var roles []*model.UserRole
	err := repo.db.WithContext(ctx).Preload("Inheritances").Preload("Permissions").Find(&roles).Error
	return roles, err
}
