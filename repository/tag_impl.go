package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// CreateTag タグを作成します
func (repo *RepositoryImpl) CreateTag(name string, restricted bool, tagType string) (*model.Tag, error) {
	t := &model.Tag{
		ID:         uuid.Must(uuid.NewV4()),
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, repo.db.Create(t).Error
}

// ChangeTagType タグの種類を変更します
func (repo *RepositoryImpl) ChangeTagType(id uuid.UUID, tagType string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return repo.db.Model(&model.Tag{ID: id}).Update("type", tagType).Error
}

// ChangeTagRestrict タグの制限を変更します
func (repo *RepositoryImpl) ChangeTagRestrict(id uuid.UUID, restrict bool) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return repo.db.Model(&model.Tag{ID: id}).Update("restricted", restrict).Error
}

// GetAllTags 全てのタグを取得します
func (repo *RepositoryImpl) GetAllTags() (result []*model.Tag, err error) {
	err = repo.db.Find(&result).Error
	return result, err
}

// GetTagByID タグを取得します
func (repo *RepositoryImpl) GetTagByID(id uuid.UUID) (*model.Tag, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	tag := &model.Tag{}
	if err := repo.db.Where(&model.Tag{ID: id}).Take(tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return tag, nil
}

// GetTagByName タグを取得します
func (repo *RepositoryImpl) GetTagByName(name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	tag := &model.Tag{}
	if err := repo.db.Where(&model.Tag{Name: name}).Take(tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return tag, nil
}

// GetOrCreateTagByName 引数のタグを取得するか、生成したものを返します。
func (repo *RepositoryImpl) GetOrCreateTagByName(name string) (*model.Tag, error) {
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

// AddUserTag ユーザーにタグを付与します
func (repo *RepositoryImpl) AddUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	ut := &model.UsersTag{
		UserID: userID,
		TagID:  tagID,
	}
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

// ChangeUserTagLock ユーザーのタグのロック状態を変更します
func (repo *RepositoryImpl) ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Model(&model.UsersTag{}).Where(&model.UsersTag{UserID: userID, TagID: tagID}).Update("is_locked", locked)
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

// DeleteUserTag ユーザーからタグを削除します
func (repo *RepositoryImpl) DeleteUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Where(&model.UsersTag{UserID: userID, TagID: tagID}).Delete(&model.UsersTag{})
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

// GetUserTag ユーザータグを取得します
func (repo *RepositoryImpl) GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error) {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return nil, ErrNotFound
	}
	ut := &model.UsersTag{}
	if err := repo.db.Preload("Tag").Where(&model.UsersTag{UserID: userID, TagID: tagID}).Take(ut).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ut, nil
}

// GetUserTagsByUserID ユーザーに付与されているタグを取得します
func (repo *RepositoryImpl) GetUserTagsByUserID(userID uuid.UUID) (tags []*model.UsersTag, err error) {
	tags = make([]*model.UsersTag, 0)
	if userID == uuid.Nil {
		return tags, nil
	}
	err = repo.db.Preload("Tag").Where(&model.UsersTag{UserID: userID}).Order("created_at").Find(&tags).Error
	return tags, err
}

// GetUsersByTag 指定したタグを持った全ユーザーを取得します
func (repo *RepositoryImpl) GetUsersByTag(tag string) (arr []*model.User, err error) {
	arr = make([]*model.User, 0)
	if len(tag) == 0 {
		return arr, nil
	}
	err = repo.db.
		Where("id IN (?)", repo.db.
			Model(&model.UsersTag{}).
			Select("users_tags.user_id").
			Joins("INNER JOIN tags ON users_tags.tag_id = tags.id AND tags.name = ?", tag).
			QueryExpr()).
		Find(&arr).
		Error
	return
}

// GetUserIDsByTag 指定したタグを持った全ユーザーのUUIDを取得します
func (repo *RepositoryImpl) GetUserIDsByTag(tag string) (arr []uuid.UUID, err error) {
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

// GetUserIDsByTagID 指定したタグを持った全ユーザーのUUIDを取得します
func (repo *RepositoryImpl) GetUserIDsByTagID(tagID uuid.UUID) (arr []uuid.UUID, err error) {
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
