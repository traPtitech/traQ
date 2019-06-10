package repository

import "github.com/traPtitech/traQ/model"

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
