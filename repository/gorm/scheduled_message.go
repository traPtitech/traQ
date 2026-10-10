package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
)

func scheduledMessageFiles(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Attachments", func(db *gorm.DB) *gorm.DB { return db.Order("position") })
}

func (repo *Repository) CreateScheduledMessage(ctx context.Context, m *model.ScheduledMessage) error {
	if m.ID == uuid.Nil || m.UserID == uuid.Nil || m.ChannelID == uuid.Nil {
		return repository.ErrNilID
	}
	m.Status = model.ScheduledMessagePending
	m.Failure = ""
	// MariaDB DATETIME does not store an offset. Persist and query reservation
	// times in UTC so clients and server replicas can use different time zones.
	m.ScheduledAt = m.ScheduledAt.UTC()
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize the per-user limit, including simultaneous API requests.
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", m.UserID).Error; err != nil {
			return convertError(err)
		}
		var count int64
		if err := tx.Model(&model.ScheduledMessage{}).Where("user_id = ? AND status IN ?", m.UserID,
			[]string{model.ScheduledMessagePending, model.ScheduledMessageFailed}).Count(&count).Error; err != nil {
			return err
		}
		if count >= repository.ScheduledMessageLimit {
			return repository.ArgError("scheduledMessages", "too many scheduled messages")
		}
		if err := validateScheduledFiles(tx, m.FileIDs(), m.UserID); err != nil {
			return err
		}
		if len(m.Attachments) > 0 {
			var reserved int64
			if err := tx.Model(&model.ScheduledMessageFile{}).Where("file_id IN ?", m.FileIDs()).Count(&reserved).Error; err != nil {
				return err
			}
			if reserved > 0 {
				return repository.ErrAlreadyExists
			}
		}
		for i := range m.Attachments {
			m.Attachments[i].ScheduledMessageID = m.ID
			m.Attachments[i].Position = i
		}
		return convertError(tx.Create(m).Error)
	})
}

func (repo *Repository) GetScheduledMessages(ctx context.Context, userID uuid.UUID) ([]*model.ScheduledMessage, error) {
	result := make([]*model.ScheduledMessage, 0)
	err := repo.db.WithContext(ctx).Scopes(scheduledMessageFiles).
		Where("user_id = ? AND status IN ?", userID, []string{model.ScheduledMessagePending, model.ScheduledMessageFailed}).
		Order("scheduled_at, id").Find(&result).Error
	return result, err
}

func (repo *Repository) GetDueScheduledMessageIDs(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := repo.db.WithContext(ctx).Model(&model.ScheduledMessage{}).
		Where("status = ? AND scheduled_at <= ?", model.ScheduledMessagePending, now.UTC()).
		Order("scheduled_at, id").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (repo *Repository) CancelScheduledMessage(ctx context.Context, id, userID uuid.UUID) (*model.ScheduledMessage, error) {
	var m model.ScheduledMessage
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Scopes(scheduledMessageFiles).
			First(&m, "id = ? AND user_id = ?", id, userID).Error; err != nil {
			return convertError(err)
		}
		if m.Status == model.ScheduledMessageSent {
			return repository.ErrAlreadyExists
		}
		// Keep the association so a canceled upload cannot be attached to another
		// reservation while the API removes it from storage.
		return tx.Model(&m).Updates(map[string]interface{}{
			"status": model.ScheduledMessageCanceled, "content": "", "draft_content": "", "failure": "",
		}).Error
	})
	return &m, err
}

func validateScheduledFiles(tx *gorm.DB, ids []uuid.UUID, userID uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var files []model.FileMeta
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Order("id").Find(&files).Error; err != nil {
		return err
	}
	if len(files) != len(ids) {
		return repository.ErrForbidden
	}
	for _, f := range files {
		if !f.CreatorID.Valid || f.CreatorID.V != userID || f.ChannelID.Valid || f.Type != model.FileTypeUserFile {
			return repository.ErrForbidden
		}
		var acl []model.FileACLEntry
		if err := tx.Where("file_id = ?", f.ID).Find(&acl).Error; err != nil {
			return err
		}
		if len(acl) != 1 || acl[0].UserID != userID || !acl[0].Allow {
			return repository.ErrForbidden
		}
	}
	return nil
}

func (repo *Repository) ProcessScheduledMessage(ctx context.Context, id uuid.UUID, validate repository.ScheduledMessageValidator) error {
	var delivered *model.Message
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m model.ScheduledMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Scopes(scheduledMessageFiles).
			First(&m, "id = ?", id).Error; err != nil {
			return convertError(err)
		}
		if m.Status != model.ScheduledMessagePending || m.ScheduledAt.After(time.Now()) {
			return nil
		}
		recipients, failure, err := validate(ctx, &m)
		if err != nil {
			return err // Transient errors leave the reservation pending.
		}
		if failure == "" {
			if err := validateScheduledFiles(tx, m.FileIDs(), m.UserID); err != nil {
				if !errors.Is(err, repository.ErrForbidden) {
					return err
				}
				failure = "attachment_unavailable"
			}
		}
		if failure != "" {
			return tx.Model(&m).Updates(map[string]interface{}{"status": model.ScheduledMessageFailed, "failure": failure}).Error
		}
		for _, f := range m.Attachments {
			if err := tx.Where("file_id = ?", f.FileID).Delete(&model.FileACLEntry{}).Error; err != nil {
				return err
			}
			for _, uid := range recipients {
				if err := tx.Create(&model.FileACLEntry{FileID: f.FileID, UserID: uid, Allow: true}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&model.FileMeta{}).Where("id = ?", f.FileID).
				Update("channel_id", optional.From(m.ChannelID)).Error; err != nil {
				return err
			}
		}
		msg := &model.Message{
			ID: uuid.Must(uuid.NewV7()), UserID: m.UserID, ChannelID: m.ChannelID,
			Text: m.Content, Stamps: []model.MessageStamp{},
		}
		if err := createMessage(tx, msg); err != nil {
			return err
		}
		if err := tx.Model(&m).Updates(map[string]interface{}{
			"status": model.ScheduledMessageSent, "content": "", "draft_content": "",
		}).Error; err != nil {
			return err
		}
		delivered = msg
		return nil
	})
	if err == nil && delivered != nil {
		repo.publishMessageCreated(delivered)
	}
	return err
}
