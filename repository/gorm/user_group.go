package gorm

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// CreateUserGroup implements UserGroupRepository interface.
func (repo *Repository) CreateUserGroup(name, description, gType string, adminID, iconFileID uuid.UUID) (*model.UserGroup, error) {
	g := &model.UserGroup{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: description,
		Icon:        iconFileID,
		Type:        gType,
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(g).Error
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}

		return tx.Create(&model.UserGroupAdmin{GroupID: g.ID, UserID: adminID}).Error
	})
	if err != nil {
		return nil, err
	}
	g.Members = make([]*model.UserGroupMember, 0)
	g.Admins = []*model.UserGroupAdmin{{GroupID: g.ID, UserID: adminID}}
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
func (repo *Repository) UpdateUserGroup(id uuid.UUID, args repository.UpdateUserGroupArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	var updated bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.First(&g, &model.UserGroup{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Name.Valid {
			changes["name"] = args.Name.V
		}
		if args.Description.Valid {
			changes["description"] = args.Description.V
		}
		if args.Type.Valid {
			changes["type"] = args.Type.V
		}
		if args.Icon.Valid {
			changes["icon"] = args.Icon.V
		}

		if len(changes) == 0 {
			return nil
		}

		updated = true
		err := tx.Model(&g).Updates(changes).Error
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	})
	if err != nil {
		return err
	}

	if updated {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupUpdated,
			Fields: hub.Fields{
				"group_id": id,
			},
		})
	}
	return nil
}

// DeleteUserGroup implements UserGroupRepository interface.
func (repo *Repository) DeleteUserGroup(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&model.UserGroupMember{GroupID: id}).Delete(&model.UserGroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where(&model.UserGroupAdmin{GroupID: id}).Delete(&model.UserGroupAdmin{}).Error; err != nil {
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

// GetUserGroup implements UserGroupRepository interface.
func (repo *Repository) GetUserGroup(id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.Scopes(userGroupPreloads).First(&g, &model.UserGroup{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserGroupByName implements UserGroupRepository interface.
func (repo *Repository) GetUserGroupByName(name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.Scopes(userGroupPreloads).First(&g, &model.UserGroup{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserBelongingGroupIDs implements UserGroupRepository interface.
func (repo *Repository) GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error) {
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
func (repo *Repository) GetAllUserGroups() ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	err := repo.db.Scopes(userGroupPreloads).Find(&groups).Error
	return groups, err
}

// AddUserToGroup implements UserGroupRepository interface.
func (repo *Repository) AddUserToGroup(userID, groupID uuid.UUID, role string) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		added   bool
		updated bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Members").First(&g, &model.UserGroup{ID: groupID}).Error; err != nil {
			return convertError(err)
		}

		if g.IsMember(userID) {
			if err := tx.Model(&model.UserGroupMember{UserID: userID, GroupID: groupID}).Update("role", role).Error; err != nil {
				return err
			}
			updated = true
		} else {
			if err := tx.Create(&model.UserGroupMember{UserID: userID, GroupID: groupID, Role: role}).Error; err != nil {
				return err
			}
			added = true
		}
		return tx.Model(&g).UpdateColumn("updated_at", time.Now()).Error
	})
	if err != nil {
		return err
	}
	if added {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupMemberAdded,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	if updated {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupMemberUpdated,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return nil
}

// AddUsersToGroup implements UserGroupRepository interface.
func (repo *Repository) AddUsersToGroup(users []model.UserGroupMember, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return repository.ErrNilID
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Members").First(&g, &model.UserGroup{ID: groupID}).Error; err != nil {
			return convertError(err)
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "group_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"role"}),
		}).Create(&users).Error; err != nil {
			return convertError(err)
		}

		for _, user := range users {
			if g.IsMember(user.UserID) {
				repo.hub.Publish(hub.Message{
					Name: event.UserGroupMemberUpdated,
					Fields: hub.Fields{
						"group_id": user.GroupID,
						"user_id":  user.UserID,
					},
				})
			} else {
				repo.hub.Publish(hub.Message{
					Name: event.UserGroupMemberAdded,
					Fields: hub.Fields{
						"group_id": user.GroupID,
						"user_id":  user.UserID,
					},
				})
			}
		}

		return tx.Model(&g).UpdateColumn("updated_at", time.Now()).Error
	})
	if err != nil {
		return err
	}

	return nil
}

// RemoveUserFromGroup implements UserGroupRepository interface.
func (repo *Repository) RemoveUserFromGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	var changed bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.Scopes(userGroupPreloads).First(&g, &model.UserGroup{ID: groupID}).Error; err != nil {
			return convertError(err)
		}

		result := tx.Delete(&model.UserGroupMember{}, &model.UserGroupMember{UserID: userID, GroupID: groupID})
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

// RemoveUsersFromGroup implements UserGroupRepository interface.
func (repo *Repository) RemoveUsersFromGroup(groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var removedUsers []model.UserGroupMember
		if err := tx.Where("group_id = ?", groupID).Find(&model.UserGroupMember{}).Error; err != nil {
			return err
		}
		for _, userID := range removedUsers {
			repo.hub.Publish(hub.Message{
				Name: event.UserGroupMemberRemoved,
				Fields: hub.Fields{
					"group_id": groupID,
					"user_id":  userID,
				},
			})
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return nil
}

// AddUserToGroupAdmin implements UserGroupRepository interface.
func (repo *Repository) AddUserToGroupAdmin(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}

	var added bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.First(&g, &model.UserGroup{ID: groupID}).Error; err != nil {
			return convertError(err)
		}

		if err := tx.Create(&model.UserGroupAdmin{UserID: userID, GroupID: groupID}).Error; err != nil {
			if gormutil.IsMySQLDuplicatedRecordErr(err) {
				return nil
			}
			return err
		}
		added = true
		return tx.Model(&g).UpdateColumn("updated_at", time.Now()).Error
	})
	if err != nil {
		return err
	}

	if added {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupAdminAdded,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return nil
}

// RemoveUserFromGroupAdmin implements UserGroupRepository interface.
func (repo *Repository) RemoveUserFromGroupAdmin(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}

	var removed bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var g model.UserGroup
		if err := tx.Scopes(userGroupPreloads).First(&g, &model.UserGroup{ID: groupID}).Error; err != nil {
			return convertError(err)
		}

		if !g.IsAdmin(userID) {
			return nil
		}
		if len(g.Admins) <= 1 {
			// Adminは必ず一人以上存在している必要がある
			return repository.ErrForbidden
		}

		if err := tx.Delete(&model.UserGroupAdmin{}, &model.UserGroupAdmin{UserID: userID, GroupID: groupID}).Error; err != nil {
			return err
		}
		removed = true
		return tx.Model(&model.UserGroup{ID: groupID}).UpdateColumn("updated_at", time.Now()).Error
	})
	if err != nil {
		return err
	}

	if removed {
		repo.hub.Publish(hub.Message{
			Name: event.UserGroupAdminRemoved,
			Fields: hub.Fields{
				"group_id": groupID,
				"user_id":  userID,
			},
		})
	}
	return nil
}

func userGroupPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Admins").
		Preload("Members")
}
