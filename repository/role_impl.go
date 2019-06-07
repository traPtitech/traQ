package repository

import "github.com/traPtitech/traQ/model"

// GetAllRoles implements UserDefinedRoleRepository interface.
func (repo *GormRepository) GetAllRoles() ([]*model.UserDefinedRole, error) {
	result := make([]*model.UserDefinedRole, 0)
	err := repo.db.
		Preload("Inheritances").
		Preload("Permissions").
		Find(&result).
		Error
	return result, err
}
