package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

func (repo *GormRepository) GetClipFolder(id uuid.UUID) (*model.ClipFolder, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	f := &model.ClipFolder{}
	if err := repo.db.Where(&model.ClipFolder{ID: id}).Take(f).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// GetClipFolders 指定したユーザーのクリップフォルダを全て取得します
func (repo *GormRepository) GetClipFolders(userID uuid.UUID) (res []*model.ClipFolder, err error) {
	res = make([]*model.ClipFolder, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	err = repo.db.Where(&model.ClipFolder{UserID: userID}).Order("name").Find(&res).Error
	return res, err
}

// CreateClipFolder クリップフォルダを作成します
func (repo *GormRepository) CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	f := &model.ClipFolder{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: userID,
		Name:   name,
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := repo.db.Create(f).Error; err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderCreated,
		Fields: hub.Fields{
			"folder_id": f.ID,
			"user_id":   userID,
		},
	})
	return f, nil
}

// UpdateClipFolderName クリップフォルダ名を更新します
func (repo *GormRepository) UpdateClipFolderName(id uuid.UUID, name string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		f  model.ClipFolder
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.ClipFolder{ID: id}).First(&f).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Where(&model.ClipFolder{ID: id}).Update("name", name).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ClipFolderUpdated,
			Fields: hub.Fields{
				"folder_id": id,
				"user_id":   f.UserID,
			},
		})
	}
	return nil
}

// DeleteClipFolder クリップフォルダを削除します
func (repo *GormRepository) DeleteClipFolder(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		f  model.ClipFolder
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.ClipFolder{ID: id}).First(&f).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Delete(&f).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ClipFolderDeleted,
			Fields: hub.Fields{
				"folder_id": id,
				"user_id":   f.UserID,
			},
		})
	}
	return nil
}

// GetClipMessage 指定したIDのクリップを取得します
func (repo *GormRepository) GetClipMessage(id uuid.UUID) (*model.Clip, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	c := &model.Clip{}
	if err := repo.db.Preload("Message").Where(&model.Clip{ID: id}).Take(c).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// GetClipMessages 指定したフォルダのクリップを全て取得します
func (repo *GormRepository) GetClipMessages(folderID uuid.UUID) (res []*model.Clip, err error) {
	res = make([]*model.Clip, 0)
	if folderID == uuid.Nil {
		return res, nil
	}
	err = repo.db.Preload("Message").Where(&model.Clip{FolderID: folderID}).Order("updated_at").Find(&res).Error
	return res, err
}

// GetClipMessagesByUser 指定したユーザーのクリップを全て取得します
func (repo *GormRepository) GetClipMessagesByUser(userID uuid.UUID) (res []*model.Clip, err error) {
	res = make([]*model.Clip, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	err = repo.db.Preload("Message").Where(&model.Clip{UserID: userID}).Order("updated_at").Find(&res).Error
	return res, err
}

// CreateClip クリップを作成します
func (repo *GormRepository) CreateClip(messageID, folderID, userID uuid.UUID) (*model.Clip, error) {
	if messageID == uuid.Nil || folderID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrNilID
	}
	c := &model.Clip{
		ID:        uuid.Must(uuid.NewV4()),
		UserID:    userID,
		MessageID: messageID,
		FolderID:  folderID,
	}
	if err := repo.db.Create(c).Error; err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipCreated,
		Fields: hub.Fields{
			"user_id": userID,
			"clip_id": c.ID,
		},
	})
	return c, nil
}

// ChangeClipFolder クリップのフォルダを変更します
func (repo *GormRepository) ChangeClipFolder(clipID, folderID uuid.UUID) error {
	if clipID == uuid.Nil || folderID == uuid.Nil {
		return ErrNilID
	}
	var (
		c  model.Clip
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Clip{ID: clipID}).First(&c).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Where(&model.Clip{ID: clipID}).Updates(&model.Clip{FolderID: folderID}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ClipMoved,
			Fields: hub.Fields{
				"clip_id": clipID,
				"user_id": c.UserID,
			},
		})
	}
	return nil
}

// DeleteClip クリップを削除します
func (repo *GormRepository) DeleteClip(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		c  model.Clip
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Clip{ID: id}).First(&c).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Delete(&model.Clip{ID: id}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ClipDeleted,
			Fields: hub.Fields{
				"clip_id": id,
				"user_id": c.UserID,
			},
		})
	}
	return nil
}
