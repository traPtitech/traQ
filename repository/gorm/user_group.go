package gorm

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// CreateUserGroup implements UserGroupRepository interface.
func (repo *Repository) CreateUserGroup(ctx context.Context, name, description, gType string, adminID, iconFileID uuid.UUID) (*model.UserGroup, error) {
	g := &model.UserGroup{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: description,
		Icon:        iconFileID,
		Type:        gType,
	}
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
	event.Publish(repo.hub, event.UserGroupCreatedPayload{
		GroupID: g.ID,
		Group:   g,
	})
	return g, nil
}

// UpdateUserGroup implements UserGroupRepository interface.
func (repo *Repository) UpdateUserGroup(ctx context.Context, id uuid.UUID, args repository.UpdateUserGroupArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	var updated bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		event.Publish(repo.hub, event.UserGroupUpdatedPayload{
			GroupID: id,
		})
	}
	return nil
}

// DeleteUserGroup implements UserGroupRepository interface.
func (repo *Repository) DeleteUserGroup(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
	event.Publish(repo.hub, event.UserGroupDeletedPayload{
		GroupID: id,
	})
	return err
}

// GetUserGroup implements UserGroupRepository interface.
func (repo *Repository) GetUserGroup(ctx context.Context, id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.WithContext(ctx).Scopes(userGroupPreloads).First(&g, &model.UserGroup{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserGroupMemberIDs implements UserGroupRepository interface.
func (repo *Repository) GetUserGroupMemberIDs(ctx context.Context, groupIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	members := make(map[uuid.UUID][]uuid.UUID)

	if len(groupIDs) == 0 {
		return members, nil
	}

	var rows []struct {
		GroupID uuid.UUID
		UserID  uuid.NullUUID
	}

	if err := repo.db.WithContext(ctx).
		Table("user_groups AS g").
		Select("g.id AS group_id, m.user_id").
		Joins("LEFT JOIN user_group_members AS m ON m.group_id = g.id").
		Where("g.id IN ?", groupIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if _, ok := members[row.GroupID]; !ok {
			members[row.GroupID] = []uuid.UUID{}
		}

		if row.UserID.Valid {
			members[row.GroupID] = append(members[row.GroupID], row.UserID.UUID)
		}
	}

	return members, nil
}

// GetUserGroupByName implements UserGroupRepository interface.
func (repo *Repository) GetUserGroupByName(ctx context.Context, name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	var g model.UserGroup
	if err := repo.db.WithContext(ctx).Scopes(userGroupPreloads).First(&g, &model.UserGroup{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return &g, nil
}

// GetUserBelongingGroupIDs implements UserGroupRepository interface.
func (repo *Repository) GetUserBelongingGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	groups := make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return groups, nil
	}
	err := repo.db.WithContext(ctx).
		Model(&model.UserGroupMember{}).
		Where(&model.UserGroupMember{UserID: userID}).
		Pluck("group_id", &groups).
		Error
	return groups, err
}

// GetAllUserGroups implements UserGroupRepository interface.
func (repo *Repository) GetAllUserGroups(ctx context.Context) ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	err := repo.db.WithContext(ctx).Scopes(userGroupPreloads).Find(&groups).Error
	return groups, err
}

// AddUserToGroup implements UserGroupRepository interface.
func (repo *Repository) AddUserToGroup(ctx context.Context, userID, groupID uuid.UUID, role string) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		added   bool
		updated bool
	)
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		event.Publish(repo.hub, event.UserGroupMemberAddedPayload{
			GroupID: groupID,
			UserID:  userID,
		})
	}
	if updated {
		event.Publish(repo.hub, event.UserGroupMemberUpdatedPayload{
			GroupID: groupID,
			UserID:  userID,
		})
	}
	return nil
}

// AddUsersToGroup implements UserGroupRepository interface.
func (repo *Repository) AddUsersToGroup(ctx context.Context, users []model.UserGroupMember, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return repository.ErrNilID
	}

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

		// NOTE: 規約 (event package のドキュメント) に反してトランザクションの中で発行している。
		// ロールバックしてもイベントは取り消されず、購読側が詰まるとロックを持ったままになる。
		// 振る舞いを変えないため、型付きイベントへの移行では現状を維持している。
		for _, user := range users {
			if g.IsMember(user.UserID) {
				event.Publish(repo.hub, event.UserGroupMemberUpdatedPayload{
					GroupID: user.GroupID,
					UserID:  user.UserID,
				})
			} else {
				event.Publish(repo.hub, event.UserGroupMemberAddedPayload{
					GroupID: user.GroupID,
					UserID:  user.UserID,
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
func (repo *Repository) RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	var changed bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		event.Publish(repo.hub, event.UserGroupMemberRemovedPayload{
			GroupID: groupID,
			UserID:  userID,
		})
	}
	return nil
}

// RemoveUsersFromGroup implements UserGroupRepository interface.
func (repo *Repository) RemoveUsersFromGroup(ctx context.Context, groupID uuid.UUID) error {
	if groupID == uuid.Nil {
		return repository.ErrNilID
	}

	var removedUsers []model.UserGroupMember

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Find(&removedUsers).Delete(&model.UserGroupMember{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, user := range removedUsers {
		event.Publish(repo.hub, event.UserGroupMemberRemovedPayload{
			GroupID: groupID,
			UserID:  user.UserID,
		})
	}
	return nil
}

// AddUserToGroupAdmin implements UserGroupRepository interface.
func (repo *Repository) AddUserToGroupAdmin(ctx context.Context, userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}

	var added bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		event.Publish(repo.hub, event.UserGroupAdminAddedPayload{
			GroupID: groupID,
			UserID:  userID,
		})
	}
	return nil
}

// RemoveUserFromGroupAdmin implements UserGroupRepository interface.
func (repo *Repository) RemoveUserFromGroupAdmin(ctx context.Context, userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}

	var removed bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		event.Publish(repo.hub, event.UserGroupAdminRemovedPayload{
			GroupID: groupID,
			UserID:  userID,
		})
	}
	return nil
}

func userGroupPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Admins").
		Preload("Members")
}
