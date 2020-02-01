package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"time"
	"unicode/utf8"
)

// CreateUserGroup implements UserGroupRepository interface.
func (repo *GormRepository) CreateUserGroup(name, description, gType string, adminID uuid.UUID) (*model.UserGroup, error) {
	g := &model.UserGroup{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        name,
		Description: description,
		Type:        gType,
		AdminUserID: adminID,
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if len(g.Name) == 0 || utf8.RuneCountInString(g.Name) > 30 {
			return ArgError("name", "Name must be non-empty and shorter than 31 characters")
		}
		// ユーザーチェック
		if exists, err := dbExists(tx, map[string]interface{}{
			"id":     g.AdminUserID,
			"status": model.UserAccountStatusActive,
			"bot":    false,
		}, (&model.User{}).TableName()); err != nil {
			return err
		} else if !exists {
			return ArgError("AdminUserID", "invalid AdminUserID")
		}
		// タイプチェック
		if utf8.RuneCountInString(g.Type) > 30 {
			return ArgError("Type", "Type must be shorter than 31 characters")
		}

		err := tx.Create(g).Error
		if isMySQLDuplicatedRecordErr(err) {
			return ErrAlreadyExists
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

// UpdateUserGroup implements UserGroupRepository interface.
func (repo *GormRepository) UpdateUserGroup(id uuid.UUID, args UpdateUserGroupNameArgs) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.First(&g, &model.UserGroup{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Name.Valid {
			if len(args.Name.String) == 0 || utf8.RuneCountInString(args.Name.String) > 30 {
				return ArgError("args.Name", "Name must be non-empty and shorter than 31 characters")
			}

			// 重複チェック
			if exists, err := dbExists(tx, &model.UserGroup{Name: args.Name.String}); err != nil {
				return err
			} else if exists {
				return ErrAlreadyExists
			}
			changes["name"] = args.Name.String
		}
		if args.Description.Valid {
			changes["description"] = args.Description.String
		}
		if args.AdminUserID.Valid {
			// ユーザーチェック
			if exists, err := dbExists(tx, map[string]interface{}{
				"id":     args.AdminUserID.UUID,
				"status": model.UserAccountStatusActive,
				"bot":    false,
			}, (&model.User{}).TableName()); err != nil {
				return err
			} else if !exists {
				return ArgError("args.AdminUserID", "invalid AdminUserID")
			}
			changes["admin_user_id"] = args.AdminUserID.UUID
		}
		if args.Type.Valid {
			if utf8.RuneCountInString(args.Type.String) > 30 {
				return ArgError("args.Type", "Type must be shorter than 31 characters")
			}
			changes["type"] = args.Type.String
		}

		if len(changes) > 0 {
			return tx.Model(&g).Updates(changes).Error
		}
		return nil
	})
	return err
}

// DeleteUserGroup implements UserGroupRepository interface.
func (repo *GormRepository) DeleteUserGroup(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&model.UserGroupMember{GroupID: id}).Delete(&model.UserGroupMember{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.UserGroup{ID: id})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
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

// GetUserGroup implements UserGroupRepository interface.
func (repo *GormRepository) GetUserGroup(id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.First(&g, &model.UserGroup{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserGroupByName implements UserGroupRepository interface.
func (repo *GormRepository) GetUserGroupByName(name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.First(&g, &model.UserGroup{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserBelongingGroupIDs implements UserGroupRepository interface.
func (repo *GormRepository) GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	groups := make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return groups, nil
	}
	err := repo.db.
		Model(&model.UserGroupMember{}).
		Where(&model.UserGroupMember{UserID: userID}).
		Pluck("group_id", &groups).
		Error
	return groups, err
}

// GetAllUserGroups implements UserGroupRepository interface.
func (repo *GormRepository) GetAllUserGroups() ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	err := repo.db.Find(&groups).Error
	return groups, err
}

// AddUserToGroup implements UserGroupRepository interface.
func (repo *GormRepository) AddUserToGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return ErrNilID
	}
	var changed bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.UserGroupMember{UserID: userID, GroupID: groupID}).Error; err != nil {
			if isMySQLDuplicatedRecordErr(err) {
				return nil
			}
			return err
		}
		changed = true
		return tx.Model(&model.UserGroup{ID: groupID}).UpdateColumn("updated_at", time.Now()).Error
	})
	if err != nil {
		return err
	}
	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupMemberAdded,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return nil
}

// RemoveUserFromGroup implements UserGroupRepository interface.
func (repo *GormRepository) RemoveUserFromGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return ErrNilID
	}
	var changed bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&model.UserGroupMember{UserID: userID, GroupID: groupID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			changed = true
			return tx.Model(&model.UserGroup{ID: groupID}).UpdateColumn("updated_at", time.Now()).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupMemberRemoved,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return nil
}

// GetUserGroupMemberIDs implements UserGroupRepository interface.
func (repo *GormRepository) GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	if groupID == uuid.Nil {
		return ids, nil
	}
	return ids, repo.db.
		Model(&model.UserGroupMember{}).
		Where(&model.UserGroupMember{GroupID: groupID}).
		Pluck("user_id", &ids).
		Error
}
