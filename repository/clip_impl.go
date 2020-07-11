package repository

import (
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/optional"
	"log"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// CreateClipFolder implements ClipRepository interface.
func (repo *GormRepository) CreateClipFolder(userID uuid.UUID, name string, description string) (*model.ClipFolder, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}

	uid := uuid.Must(uuid.NewV4())
	clipFolder := &model.ClipFolder{
		ID:          uid,
		Description: description,
		Name:        name,
		OwnerID:     userID,
	}
	if err := repo.db.Create(clipFolder).Error; err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderCreated,
		Fields: hub.Fields{
			"user_id":        clipFolder.OwnerID,
			"clip_folder_id": clipFolder.ID,
			"clip_folder":    clipFolder,
		},
	})

	return clipFolder, nil
}

// UpdateClipFolder implements ClipRepository interface.
func (repo *GormRepository) UpdateClipFolder(folderID uuid.UUID, name optional.String, description optional.String) error {
	if folderID == uuid.Nil {
		return ErrNilID
	}

	changes := map[string]interface{}{}

	if name.Valid {
		changes["name"] = name.String
	}

	if description.Valid {
		changes["description"] = description.String
	}

	var (
		old model.ClipFolder
		new model.ClipFolder
		ok  bool
	)

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&old, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}

		// update
		log.Println(changes)
		if len(changes) > 0 {
			if err := tx.Model(&old).Updates(changes).Error; err != nil {
				return err
			}
		}
		ok = true
		return tx.Where(&model.ClipFolder{ID: folderID}).First(&new).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ClipFolderUpdated,
			Fields: hub.Fields{
				"user_id":         old.OwnerID,
				"clip_folder_id":  folderID,
				"old_clip_folder": &old,
				"clip_folder":     &new,
			},
		})
	}
	return nil
}

// DeleteClipFolder implements ClipRepository interface.
func (repo *GormRepository) DeleteClipFolder(folderID uuid.UUID) error {
	if folderID == uuid.Nil {
		return ErrNilID
	}
	var cf model.ClipFolder
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&cf, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(&model.ClipFolderMessage{FolderID: folderID}).Delete(&model.ClipFolder{ID: folderID}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderDeleted,
		Fields: hub.Fields{
			"user_id":        cf.OwnerID,
			"clip_folder_id": folderID,
			"clip_folder":    &cf,
		},
	})
	return nil
}

// DeleteClipFolderMessage implements ClipRepository interface.
func (repo *GormRepository) DeleteClipFolderMessage(folderID, messageID uuid.UUID) error {
	if folderID == uuid.Nil || messageID == uuid.Nil {
		return ErrNilID
	}
	var (
		cf  model.ClipFolder
		cfm model.ClipFolderMessage
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// フォルダ存在チェック
		if err := tx.First(&cf, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}

		// クリップメッセージ存在チェック
		if err := tx.First(&cfm, &model.ClipFolderMessage{MessageID: messageID, FolderID: folderID}).Error; err != nil && !gorm.IsRecordNotFoundError(err) {
			return err
		}
		return tx.Delete(&model.ClipFolderMessage{MessageID: messageID, FolderID: folderID}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderMessageDeleted,
		Fields: hub.Fields{
			"user_id":                cf.OwnerID,
			"clip_folder_id":         cf.ID,
			"clip_folder_message_id": messageID,
			"clip_folder_message":    &cfm,
		},
	})
	return nil
}

// AddClipFolderMessage implements ClipRepository interface.
func (repo *GormRepository) AddClipFolderMessage(folderID, messageID uuid.UUID) (*model.ClipFolderMessage, error) {
	if folderID == uuid.Nil || messageID == uuid.Nil {
		return nil, ErrNilID
	}

	cfm := &model.ClipFolderMessage{
		FolderID:  folderID,
		MessageID: messageID,
	}
	var cf model.ClipFolder

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// フォルダ存在チェック
		if err := tx.First(&cf, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}

		// 名前重複チェック
		if exists, err := gormutil.RecordExists(tx, &model.ClipFolderMessage{FolderID: folderID, MessageID: messageID}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}

		if err := tx.Create(cfm).Error; err != nil {
			return err
		}

		return tx.Scopes(messagePreloads).First(&cfm.Message, model.Message{ID: cfm.MessageID}).Error
	})
	if err != nil {
		return cfm, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.ClipFolderMessageAdded,
		Fields: hub.Fields{
			"user_id":                cf.OwnerID,
			"clip_folder_id":         cf.ID,
			"clip_folder_message_id": messageID,
			"clip_folder_message":    cfm,
		},
	})

	return cfm, nil
}

// GetClipFoldersByUserID implements ClipRepository interface.
func (repo *GormRepository) GetClipFoldersByUserID(userID uuid.UUID) ([]*model.ClipFolder, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}

	clipFolders := make([]*model.ClipFolder, 0)

	if err := repo.db.Find(&clipFolders, "owner_id=?", userID).Error; err != nil {
		return nil, convertError(err)
	}

	return clipFolders, nil
}

// GetClipFolder implements ClipRepository interface.
func (repo *GormRepository) GetClipFolder(folderID uuid.UUID) (*model.ClipFolder, error) {
	if folderID == uuid.Nil {
		return nil, ErrNilID
	}
	clipFolder := &model.ClipFolder{}

	if err := repo.db.First(clipFolder, &model.ClipFolder{ID: folderID}).Error; err != nil {
		return nil, convertError(err)
	}

	return clipFolder, nil
}

// GetClipFolderMessages implements ClipRepository interface.
func (repo *GormRepository) GetClipFolderMessages(folderID uuid.UUID, query ClipFolderMessageQuery) (messages []*model.ClipFolderMessage, more bool, err error) {
	if folderID == uuid.Nil {
		return nil, false, ErrNilID
	}
	messages = make([]*model.ClipFolderMessage, 0)

	// フォルダ存在チェック
	if exists, err := gormutil.RecordExists(repo.db, &model.ClipFolder{ID: folderID}); err != nil {
		return nil, false, err
	} else if !exists {
		return nil, false, ErrNotFound
	}

	tx := repo.db
	tx = tx.Where("folder_id=?", folderID).Scopes(clipPreloads)

	if query.Asc {
		tx = tx.Order("created_at")
	} else {
		tx = tx.Order("created_at DESC")
	}

	if query.Offset > 0 {
		tx = tx.Offset(query.Offset)
	}

	if query.Limit > 0 {
		err = tx.Limit(query.Limit + 1).Find(&messages).Error
		if len(messages) > query.Limit {
			return messages[:len(messages)-1], true, err
		}
	} else {
		err = tx.Find(&messages).Error
	}
	return messages, false, err
}

// GetMessageClips implements ClipRepository interface.
func (repo *GormRepository) GetMessageClips(userID, messageID uuid.UUID) ([]*model.ClipFolderMessage, error) {
	clips := make([]*model.ClipFolderMessage, 0)
	if userID == uuid.Nil || messageID == uuid.Nil {
		return clips, nil
	}

	err := repo.db.
		Joins("INNER JOIN clip_folders ON clip_folders.id = clip_folder_messages.folder_id").
		Where("clip_folders.owner_id = ? AND clip_folder_messages.message_id = ?", userID, messageID).
		Find(&clips).
		Error
	return clips, err
}

func clipPreloads(db *gorm.DB) *gorm.DB {
	return db.Preload("Message").Preload("Message.Stamps").Preload("Message.Pin")
}
