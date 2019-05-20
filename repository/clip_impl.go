package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"unicode/utf8"
)

// GetClipFolder implements ClipRepository interface.
func (repo *GormRepository) GetClipFolder(id uuid.UUID) (*model.ClipFolder, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	f := &model.ClipFolder{}
	if err := repo.db.Where(&model.ClipFolder{ID: id}).Take(f).Error; err != nil {
		return nil, convertError(err)
	}
	return f, nil
}

// GetClipFolders implements ClipRepository interface.
func (repo *GormRepository) GetClipFolders(userID uuid.UUID) (res []*model.ClipFolder, err error) {
	res = make([]*model.ClipFolder, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	err = repo.db.Where(&model.ClipFolder{UserID: userID}).Order("name").Find(&res).Error
	return res, err
}

// CreateClipFolder implements ClipRepository interface.
func (repo *GormRepository) CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	if len(name) == 0 || utf8.RuneCountInString(name) > 30 {
		return nil, ArgError("name", "Name must be non-empty and shorter than 31 characters")
	}

	f := &model.ClipFolder{
		ID:     uuid.Must(uuid.NewV4()),
		UserID: userID,
		Name:   name,
	}
	err := repo.transact(func(tx *gorm.DB) error {
		// 重複チェック
		if exists, err := dbExists(tx, &model.ClipFolder{UserID: userID, Name: name}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}
		return repo.db.Create(f).Error
	})
	if err != nil {
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

// UpdateClipFolderName implements ClipRepository interface.
func (repo *GormRepository) UpdateClipFolderName(id uuid.UUID, name string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	if len(name) == 0 || utf8.RuneCountInString(name) > 30 {
		return ArgError("name", "Name must be non-empty and shorter than 31 characters")
	}

	var f model.ClipFolder
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.ClipFolder{ID: id}).First(&f).Error; err != nil {
			return convertError(err)
		}

		// 重複チェック
		if exists, err := dbExists(tx, &model.ClipFolder{UserID: f.UserID, Name: name}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}

		return tx.Where(&model.ClipFolder{ID: id}).Update("name", name).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderUpdated,
		Fields: hub.Fields{
			"folder_id": id,
			"user_id":   f.UserID,
		},
	})
	return nil
}

// DeleteClipFolder implements ClipRepository interface.
func (repo *GormRepository) DeleteClipFolder(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}

	var f model.ClipFolder
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.ClipFolder{ID: id}).First(&f).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(&f).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderDeleted,
		Fields: hub.Fields{
			"folder_id": id,
			"user_id":   f.UserID,
		},
	})
	return nil
}

// GetClipMessage implements ClipRepository interface.
func (repo *GormRepository) GetClipMessage(id uuid.UUID) (*model.Clip, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	c := &model.Clip{}
	if err := repo.db.Scopes(clipPreloads).Where(&model.Clip{ID: id}).Take(c).Error; err != nil {
		return nil, convertError(err)
	}
	return c, nil
}

// GetClipMessages implements ClipRepository interface.
func (repo *GormRepository) GetClipMessages(folderID uuid.UUID) (res []*model.Clip, err error) {
	res = make([]*model.Clip, 0)
	if folderID == uuid.Nil {
		return res, nil
	}
	err = repo.db.
		Scopes(clipPreloads).
		Where(&model.Clip{FolderID: folderID}).
		Order("updated_at").
		Find(&res).
		Error
	return res, err
}

// GetClipMessagesByUser implements ClipRepository interface.
func (repo *GormRepository) GetClipMessagesByUser(userID uuid.UUID) (res []*model.Clip, err error) {
	res = make([]*model.Clip, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	err = repo.db.
		Scopes(clipPreloads).
		Where(&model.Clip{UserID: userID}).
		Order("updated_at").
		Find(&res).
		Error
	return res, err
}

// CreateClip implements ClipRepository interface.
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

// ChangeClipFolder implements ClipRepository interface.
func (repo *GormRepository) ChangeClipFolder(clipID, folderID uuid.UUID) error {
	if clipID == uuid.Nil || folderID == uuid.Nil {
		return ErrNilID
	}

	var c model.Clip
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&c, &model.Clip{ID: clipID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Where(&model.Clip{ID: clipID}).Updates(&model.Clip{FolderID: folderID}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipMoved,
		Fields: hub.Fields{
			"clip_id": clipID,
			"user_id": c.UserID,
		},
	})
	return nil
}

// DeleteClip implements ClipRepository interface.
func (repo *GormRepository) DeleteClip(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}

	var c model.Clip
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&c, &model.Clip{ID: id}).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(&model.Clip{ID: id}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipDeleted,
		Fields: hub.Fields{
			"clip_id": id,
			"user_id": c.UserID,
		},
	})
	return nil
}

func clipPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Message").
		Preload("Message.Stamps", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_at")
		}).
		Preload("Message.Pin")
}
