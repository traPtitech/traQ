package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
)

// GetFileMetas implements FileRepository interface.
func (repo *GormRepository) GetFileMetas(q FilesQuery) (result []*model.FileMeta, more bool, err error) {
	files := make([]*model.FileMeta, 0)
	tx := repo.db.Where("files.type = ?", q.Type.String())

	if q.ChannelID.Valid {
		if q.ChannelID.UUID == uuid.Nil {
			tx = tx.Where("files.channel_id IS NULL")
		} else {
			tx = tx.Where("files.channel_id = ?", q.ChannelID.UUID)
		}
	}
	if q.UploaderID.Valid {
		if q.UploaderID.UUID == uuid.Nil {
			tx = tx.Where("files.creator_id IS NULL")
		} else {
			tx = tx.Where("files.creator_id = ?", q.UploaderID.UUID)
		}
	}

	if q.Inclusive {
		if q.Since.Valid {
			tx = tx.Where("files.created_at >= ?", q.Since.Time)
		}
		if q.Until.Valid {
			tx = tx.Where("files.created_at <= ?", q.Until.Time)
		}
	} else {
		if q.Since.Valid {
			tx = tx.Where("files.created_at > ?", q.Since.Time)
		}
		if q.Until.Valid {
			tx = tx.Where("files.created_at < ?", q.Until.Time)
		}
	}

	if q.Asc {
		tx = tx.Order("files.created_at")
	} else {
		tx = tx.Order("files.created_at DESC")
	}

	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	if q.Limit > 0 {
		err = tx.Limit(q.Limit + 1).Find(&files).Error
		if len(files) > q.Limit {
			return files[:len(files)-1], true, err
		}
	} else {
		err = tx.Find(&files).Error
	}
	return files, false, err
}

func (repo *GormRepository) SaveFileMeta(meta *model.FileMeta, acl []*model.FileACLEntry) error {
	if meta == nil || meta.ID == uuid.Nil {
		return ErrNilID
	}
	return repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(meta).Error; err != nil {
			return err
		}
		for _, entry := range acl {
			entry.FileID = meta.ID
			if err := tx.Create(entry).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetFileMeta implements FileRepository interface.
func (repo *GormRepository) GetFileMeta(fileID uuid.UUID) (*model.FileMeta, error) {
	if fileID == uuid.Nil {
		return nil, ErrNotFound
	}
	f := &model.FileMeta{}
	if err := repo.db.First(f, &model.FileMeta{ID: fileID}).Error; err != nil {
		return nil, convertError(err)
	}
	return f, nil
}

// DeleteFileMeta implements FileRepository interface.
func (repo *GormRepository) DeleteFileMeta(fileID uuid.UUID) error {
	if fileID == uuid.Nil {
		return ErrNilID
	}
	return repo.db.Delete(&model.FileMeta{ID: fileID}).Error
}

// IsFileAccessible implements FileRepository interface.
func (repo *GormRepository) IsFileAccessible(fileID, userID uuid.UUID) (bool, error) {
	var result struct {
		Allow int
		Deny  int
	}
	err := repo.db.
		Model(&model.FileACLEntry{}).
		Select("COUNT(allow = TRUE OR NULL) AS allow, COUNT(allow = FALSE OR NULL) AS deny").
		Where("file_id = ? AND user_id IN (?)", fileID, []uuid.UUID{userID, uuid.Nil}).
		Scan(&result).
		Error
	if err != nil {
		return false, err
	}
	return result.Allow > 0 && result.Deny == 0, nil
}
