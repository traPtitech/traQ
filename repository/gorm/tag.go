package gorm

import (
	"context"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// GetTagByID implements TagRepository interface.
func (repo *Repository) GetTagByID(ctx context.Context, id uuid.UUID) (*model.Tag, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	tag := &model.Tag{}
	if err := repo.db.WithContext(ctx).Take(tag, &model.Tag{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return tag, nil
}

// GetOrCreateTag implements TagRepository interface.
func (repo *Repository) GetOrCreateTag(ctx context.Context, name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	if utf8.RuneCountInString(name) > 30 {
		return nil, repository.ArgError("name", "tag must be non-empty and shorter than 31 characters")
	}
	tag := &model.Tag{}
	err := repo.db.WithContext(ctx).
		Where(&model.Tag{Name: name}).
		Attrs(&model.Tag{ID: uuid.Must(uuid.NewV7())}).
		FirstOrCreate(tag).
		Error
	return tag, err
}

// AddUserTag implements TagRepository interface.
func (repo *Repository) AddUserTag(ctx context.Context, userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	ut := &model.UsersTag{
		UserID: userID,
		TagID:  tagID,
	}
	// TODO タグの存在確認
	if err := repo.db.WithContext(ctx).Create(ut).Error; err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserTagAdded,
		Fields: hub.Fields{
			"user_id": userID,
			"tag_id":  tagID,
		},
	})
	return nil
}

// ChangeUserTagLock implements TagRepository interface.
func (repo *Repository) ChangeUserTagLock(ctx context.Context, userID, tagID uuid.UUID, locked bool) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}

	result := repo.db.WithContext(ctx).Model(&model.UsersTag{UserID: userID, TagID: tagID}).Update("is_locked", locked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserTagUpdated,
		Fields: hub.Fields{
			"user_id": userID,
			"tag_id":  tagID,
		},
	})
	return nil
}

// DeleteUserTag implements TagRepository interface.
func (repo *Repository) DeleteUserTag(ctx context.Context, userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.WithContext(ctx).Delete(&model.UsersTag{}, &model.UsersTag{UserID: userID, TagID: tagID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserTagRemoved,
			Fields: hub.Fields{
				"user_id": userID,
				"tag_id":  tagID,
			},
		})
	}
	return nil
}

// GetUserTag implements TagRepository interface.
func (repo *Repository) GetUserTag(ctx context.Context, userID, tagID uuid.UUID) (model.UserTag, error) {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	ut := &model.UsersTag{}
	if err := repo.db.WithContext(ctx).Preload("Tag").Take(ut, &model.UsersTag{UserID: userID, TagID: tagID}).Error; err != nil {
		return nil, convertError(err)
	}
	return ut, nil
}

// GetUserTagsByUserID implements TagRepository interface.
func (repo *Repository) GetUserTagsByUserID(ctx context.Context, userID uuid.UUID) (tags []model.UserTag, err error) {
	var tmp []*model.UsersTag
	if userID == uuid.Nil {
		return tags, nil
	}
	err = repo.db.WithContext(ctx).
		Preload("Tag").
		Where(&model.UsersTag{UserID: userID}).
		Order("created_at").
		Find(&tmp).
		Error
	tags = make([]model.UserTag, len(tmp))
	for i, tag := range tmp {
		tags[i] = tag
	}
	return tags, err
}

// GetUserIDsByTagID implements TagRepository interface.
func (repo *Repository) GetUserIDsByTagID(ctx context.Context, tagID uuid.UUID) (arr []uuid.UUID, err error) {
	arr = make([]uuid.UUID, 0)
	if tagID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.WithContext(ctx).
		Model(&model.UsersTag{}).
		Where(&model.UsersTag{TagID: tagID}).
		Pluck("user_id", &arr).
		Error
	return arr, err
}
