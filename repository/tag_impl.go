package repository

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"unicode/utf8"
)

// CreateTag implements TagRepository interface.
func (repo *GormRepository) CreateTag(name string, restricted bool, tagType string) (*model.Tag, error) {
	// 名前チェック
	if len(name) == 0 || utf8.RuneCountInString(name) > 30 {
		return nil, ArgError("name", "Name must be non-empty and shorter than 31 characters")
	}
	// TODO タグの存在確認

	t := &model.Tag{
		ID:         uuid.Must(uuid.NewV4()),
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
	}
	return t, repo.db.Create(t).Error
}

// ChangeTagType implements TagRepository interface.
func (repo *GormRepository) ChangeTagType(id uuid.UUID, tagType string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	// TODO タグの存在確認
	return repo.db.Model(&model.Tag{ID: id}).Update("type", tagType).Error
}

// ChangeTagRestrict implements TagRepository interface.
func (repo *GormRepository) ChangeTagRestrict(id uuid.UUID, restrict bool) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	// TODO タグの存在確認
	return repo.db.Model(&model.Tag{ID: id}).Update("restricted", restrict).Error
}

// GetTagByID implements TagRepository interface.
func (repo *GormRepository) GetTagByID(id uuid.UUID) (*model.Tag, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	tag := &model.Tag{}
	if err := repo.db.Take(tag, &model.Tag{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return tag, nil
}

// GetTagByName implements TagRepository interface.
func (repo *GormRepository) GetTagByName(name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	tag := &model.Tag{}
	if err := repo.db.Take(tag, &model.Tag{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return tag, nil
}

// GetOrCreateTagByName implements TagRepository interface.
func (repo *GormRepository) GetOrCreateTagByName(name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	tag := &model.Tag{}
	err := repo.db.
		Where(&model.Tag{Name: name}).
		Attrs(&model.Tag{ID: uuid.Must(uuid.NewV4())}).
		FirstOrCreate(tag).
		Error
	return tag, err
}

// AddUserTag implements TagRepository interface.
func (repo *GormRepository) AddUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	ut := &model.UsersTag{
		UserID: userID,
		TagID:  tagID,
	}
	// TODO タグの存在確認
	if err := repo.db.Create(ut).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return ErrAlreadyExists
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
func (repo *GormRepository) ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	// TODO タグの存在確認
	result := repo.db.Model(&model.UsersTag{UserID: userID, TagID: tagID}).Update("is_locked", locked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserTagUpdated,
			Fields: hub.Fields{
				"user_id": userID,
				"tag_id":  tagID,
			},
		})
	}
	return nil
}

// DeleteUserTag implements TagRepository interface.
func (repo *GormRepository) DeleteUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.UsersTag{UserID: userID, TagID: tagID})
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
func (repo *GormRepository) GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error) {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return nil, ErrNotFound
	}
	ut := &model.UsersTag{}
	if err := repo.db.Preload("Tag").Take(ut, &model.UsersTag{UserID: userID, TagID: tagID}).Error; err != nil {
		return nil, convertError(err)
	}
	return ut, nil
}

// GetUserTagsByUserID implements TagRepository interface.
func (repo *GormRepository) GetUserTagsByUserID(userID uuid.UUID) (tags []*model.UsersTag, err error) {
	tags = make([]*model.UsersTag, 0)
	if userID == uuid.Nil {
		return tags, nil
	}
	err = repo.db.Preload("Tag").Where(&model.UsersTag{UserID: userID}).Order("created_at").Find(&tags).Error
	return tags, err
}

// GetUserIDsByTag implements TagRepository interface.
func (repo *GormRepository) GetUserIDsByTag(tag string) (arr []uuid.UUID, err error) {
	arr = make([]uuid.UUID, 0)
	if len(tag) == 0 {
		return arr, nil
	}
	err = repo.db.
		Model(&model.UsersTag{}).
		Where("tag_id = ?", repo.db.
			Model(&model.Tag{}).
			Select("id").
			Where(&model.Tag{Name: tag}).
			SubQuery()).
		Pluck("user_id", &arr).
		Error
	return arr, err
}

// GetUserIDsByTagID implements TagRepository interface.
func (repo *GormRepository) GetUserIDsByTagID(tagID uuid.UUID) (arr []uuid.UUID, err error) {
	arr = make([]uuid.UUID, 0)
	if tagID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.
		Model(&model.UsersTag{}).
		Where(&model.UsersTag{TagID: tagID}).
		Pluck("user_id", &arr).
		Error
	return arr, err
}
