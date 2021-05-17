package repository

import "github.com/traPtitech/traQ/model"

// CreateUserRoles implements UserRoleRepository interface.
func (repo *GormRepository) CreateUserRoles(roles ...*model.UserRole) error {
	return repo.db.Create(roles).Error
}

// GetAllUserRoles implements UserRoleRepository interface.
func (repo *GormRepository) GetAllUserRoles() ([]*model.UserRole, error) {
	var roles []*model.UserRole
	err := repo.db.Preload("Inheritances").Preload("Permissions").Find(&roles).Error
	return roles, err
}
