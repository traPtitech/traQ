package repository

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
)

// CreateClipFolder implements ClipRepository interface.
func (repo *GormRepository) CreateClipFolder(userID uuid.UUID, name string, description string) (*model.ClipFolder, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}

	// 名前チェック
	if err := validation.Validate(name, validator.ClipFolderNameRuleRequired...); err != nil {
		return nil, ArgError("name", "Name must be 1-32 characters of a-zA-Z0-9_-")
	}
	// descriptionチェック
	if err := validation.Validate(name, validator.ClipFolderDescriptionRule...); err != nil {
		return nil, ArgError("description", "description must be less than 1000 characters")
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

	return clipFolder, nil
}

// UpdateClipFolder implements ClipRepository interface.
func (repo *GormRepository) UpdateClipFolder(folderID uuid.UUID, name null.String, description null.String) error {
	if folderID == uuid.Nil {
		return ErrNilID
	}

	changes := map[string]interface{}{}

	// 名前チェック
	if name.Valid {
		if err := validation.Validate(name, validator.ClipFolderNameRuleRequired...); err != nil {
			return ArgError("name", "Name must be 1-30")
		}
		changes["name"] = name
	}
	// descriptionチェック
	if description.Valid {
		if err := validation.Validate(name, validator.ClipFolderDescriptionRule...); err != nil {
			return ArgError("description", "description must be less than 1000 characters")
		}
		changes["description"] = description
	}

	var (
		old model.ClipFolder
	)

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&old, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}

		// update
		if len(changes) > 0 {
			if err := tx.Model(&old).Updates(changes).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClipFolder implements ClipRepository interface.
func (repo *GormRepository) DeleteClipFolder(folderID uuid.UUID) error {
	if folderID == uuid.Nil {
		return ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var cf model.ClipFolder
		if err := tx.First(&cf, &model.ClipFolder{ID: folderID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(&model.ClipFolder{ID: folderID}).Error
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClipFolderMessage implements ClipRepository interface.
func (repo *GormRepository) DeleteClipFolderMessage(folderID, messageID uuid.UUID) error {
	if folderID == uuid.Nil || messageID == uuid.Nil {
		return ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var cfm model.ClipFolderMessage
		if err := tx.First(&cfm, &model.ClipFolderMessage{MessageID: messageID, FolderID: folderID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(&model.ClipFolderMessage{MessageID: messageID, FolderID: folderID}).Error
	})
	if err != nil {
		return err
	}
	return nil
}

// AddClipFolderMesssage implements ClipRepository interface.
func (repo *GormRepository) AddClipFolderMessage(folderID, messageID uuid.UUID) (*model.ClipFolderMessage, error) {
	if folderID == uuid.Nil || messageID == uuid.Nil {
		return nil, ErrNilID
	}

	cfm := &model.ClipFolderMessage{
		FolderID:  folderID,
		MessageID: messageID,
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前重複チェック
		if exists, err := dbExists(tx, &model.ClipFolderMessage{FolderID: folderID, MessageID: messageID}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}
		return tx.Create(cfm).Error
	})
	if err != nil {
		return cfm, err
	}

	return cfm, nil
}

// GetClipFolderByUserID implements ClipRepository interface.
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

	tx := repo.db
	tx = tx.Where("folder_id=?", folderID).Scopes(clipPreloads)

	if query.Asc {
		tx = tx.Order("created_at")
	} else {
		tx = tx.Order("created_at DESC")
	}

	if query.Offset > 0 {
		tx.Offset(query.Offset)
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

func clipPreloads(db *gorm.DB) *gorm.DB {
	return db.Preload("Message").Preload("Message.Stamps", func(db *gorm.DB) *gorm.DB {
		return db.Order("updated_at")
	}).Preload("Message.Pin")
}
