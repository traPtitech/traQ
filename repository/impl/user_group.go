package impl

import (
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// CreateUserGroup ユーザーグループを作成します
func (repo *RepositoryImpl) CreateUserGroup(name, description string, adminID uuid.UUID) (*model.UserGroup, error) {
	g := &model.UserGroup{
		ID:          uuid.NewV4(),
		Name:        name,
		Description: description,
		AdminUserID: adminID,
	}
	err := repo.transact(func(tx *gorm.DB) error {
		err := tx.Create(g).Error
		if isMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserGroupCreated,
		Fields: hub.Fields{
			"group_id": g.ID,
			"group":    g,
		},
	})
	return g, nil
}

// UpdateUserGroup ユーザーグループを更新します
func (repo *RepositoryImpl) UpdateUserGroup(id uuid.UUID, args repository.UpdateUserGroupNameArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.Where(&model.UserGroup{ID: id}).First(&g).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
		}
		if len(args.Name) > 0 {
			g.Name = args.Name
		}
		if args.Description.Valid {
			g.Description = args.Description.String
		}
		if args.AdminUserID.Valid {
			g.AdminUserID = args.AdminUserID.UUID
		}

		err := tx.Save(&g).Error
		if isMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	})
	return err
}

// DeleteUserGroup ユーザーグループを削除します
func (repo *RepositoryImpl) DeleteUserGroup(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.UserGroupMember{GroupID: id}).Delete(&model.UserGroupMember{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.UserGroup{ID: id})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return repository.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserGroupDeleted,
		Fields: hub.Fields{
			"group_id": id,
		},
	})
	return err
}

// GetUserGroup ユーザーグループを取得します
func (repo *RepositoryImpl) GetUserGroup(id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.Where(&model.UserGroup{ID: id}).First(&g).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

// GetUserGroupByName ユーザーグループを取得します
func (repo *RepositoryImpl) GetUserGroupByName(name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.Where(&model.UserGroup{Name: name}).First(&g).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

// GetUserBelongingGroups ユーザーが所属しているグループを取得します
func (repo *RepositoryImpl) GetUserBelongingGroups(userID uuid.UUID) ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	if userID == uuid.Nil {
		return groups, nil
	}
	err := repo.db.
		Where("id IN (?)", repo.db.
			Model(&model.UserGroupMember{}).
			Select("group_id").
			Where(&model.UserGroupMember{UserID: userID}).
			QueryExpr()).
		Find(&groups).
		Error
	return groups, err
}

// GetAllUserGroups 全てのグループを取得します
func (repo *RepositoryImpl) GetAllUserGroups() ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	err := repo.db.Find(&groups).Error
	return groups, err
}

// AddUserToGroup グループにユーザーを追加します
func (repo *RepositoryImpl) AddUserToGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.Create(&model.UserGroupMember{UserID: userID, GroupID: groupID}).Error
	if err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return nil
		}
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserGroupMemberAdded,
		Fields: hub.Fields{
			"group_id": groupID,
			"user_id":  userID,
		},
	})
	return nil
}

// RemoveUserFromGroup グループからユーザーを削除します
func (repo *RepositoryImpl) RemoveUserFromGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Where(&model.UserGroupMember{UserID: userID, GroupID: groupID}).Delete(&model.UserGroupMember{})
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupMemberRemoved,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return result.Error
}

// GetUserGroupMemberIDs グループのメンバーのUUIDを取得します
func (repo *RepositoryImpl) GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	if groupID == uuid.Nil {
		return ids, nil
	}
	err := repo.db.
		Model(&model.UserGroupMember{}).
		Where(&model.UserGroupMember{GroupID: groupID}).
		Pluck("user_id", &ids).
		Error
	return ids, err
}
