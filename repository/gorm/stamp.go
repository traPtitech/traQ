package gorm

import (
	"sync"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/validator"
)

type stampRepository struct {
	stamps       map[uuid.UUID]*model.Stamp
	allJSON      []byte
	unicodeJSON  []byte
	originalJSON []byte
	updatedAt    time.Time
	sync.RWMutex
}

func makeStampRepository(stamps []*model.Stamp) *stampRepository {
	r := &stampRepository{
		stamps:    make(map[uuid.UUID]*model.Stamp, len(stamps)),
		updatedAt: time.Now(),
	}
	for _, s := range stamps {
		r.stamps[s.ID] = s
	}

	r.regenerateJSON()
	return r
}

func (r *stampRepository) add(s *model.Stamp) {
	r.stamps[s.ID] = s
	r.updatedAt = time.Now()
	r.regenerateJSON()
}

func (r *stampRepository) update(s *model.Stamp) {
	r.stamps[s.ID] = s
	r.updatedAt = time.Now()
	r.regenerateJSON()
}

func (r *stampRepository) delete(id uuid.UUID) {
	delete(r.stamps, id)
	r.updatedAt = time.Now()
	r.regenerateJSON()
}

func (r *stampRepository) regenerateJSON() {
	arrOriginal := make([]*model.Stamp, 0, len(r.stamps))
	arrUnicode := make([]*model.Stamp, 0, len(r.stamps))
	arrAll := make([]*model.Stamp, 0, len(r.stamps))
	for _, stamp := range r.stamps {
		arrAll = append(arrAll, stamp)
		if stamp.IsUnicode {
			arrOriginal = append(arrOriginal, stamp)
		} else {
			arrUnicode = append(arrUnicode, stamp)
		}
	}

	b, err := jsoniter.ConfigFastest.Marshal(arrUnicode)
	if err != nil {
		panic(err)
	}
	r.unicodeJSON = b

	b, err = jsoniter.ConfigFastest.Marshal(arrOriginal)
	if err != nil {
		panic(err)
	}
	r.originalJSON = b

	b, err = jsoniter.ConfigFastest.Marshal(arrAll)
	if err != nil {
		panic(err)
	}
	r.allJSON = b
}

func (r *stampRepository) GetStamp(id uuid.UUID) (s *model.Stamp, ok bool) {
	r.RLock()
	defer r.RUnlock()
	s, ok = r.stamps[id]
	return
}

func (r *stampRepository) CheckIDs(ids []uuid.UUID) bool {
	r.RLock()
	defer r.RUnlock()
	for _, id := range ids {
		if _, ok := r.stamps[id]; !ok {
			return false
		}
	}
	return true
}

// CreateStamp implements StampRepository interface.
func (repo *Repository) CreateStamp(args repository.CreateStampArgs) (s *model.Stamp, err error) {
	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      args.Name,
		FileID:    args.FileID,
		CreatorID: args.CreatorID, // uuid.Nilを許容する
		IsUnicode: args.IsUnicode,
	}

	if repo.stamps != nil {
		repo.stamps.Lock()
		defer repo.stamps.Unlock()
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if err := vd.Validate(stamp.Name, validator.StampNameRuleRequired...); err != nil {
			return repository.ArgError("name", "Name must be 1-32 characters of a-zA-Z0-9_-")
		}
		// 名前重複チェック
		if exists, err := gormutil.RecordExists(tx, &model.Stamp{Name: stamp.Name}); err != nil {
			return err
		} else if exists {
			return repository.ErrAlreadyExists
		}
		// ファイル存在チェック
		if stamp.FileID == uuid.Nil {
			return repository.ArgError("fileID", "FileID's file is not found")
		}
		if exists, err := gormutil.RecordExists(tx, &model.FileMeta{ID: stamp.FileID}); err != nil {
			return err
		} else if !exists {
			return repository.ArgError("fileID", "fileID's file is not found")
		}

		return tx.Create(stamp).Error
	})
	if err != nil {
		return nil, err
	}

	if repo.stamps != nil {
		repo.stamps.add(stamp)
	}

	repo.hub.Publish(hub.Message{
		Name: event.StampCreated,
		Fields: hub.Fields{
			"stamp":    stamp,
			"stamp_id": stamp.ID,
		},
	})
	return stamp, nil
}

// UpdateStamp implements StampRepository interface.
func (repo *Repository) UpdateStamp(id uuid.UUID, args repository.UpdateStampArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	if repo.stamps != nil {
		repo.stamps.Lock()
		defer repo.stamps.Unlock()
	}

	var s model.Stamp
	changes := map[string]interface{}{}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&s, &model.Stamp{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if args.Name.Valid && s.Name != args.Name.String {
			if err := vd.Validate(args.Name.String, validator.StampNameRuleRequired...); err != nil {
				return repository.ArgError("args.Name", "Name must be 1-32 characters of a-zA-Z0-9_-")
			}

			// 重複チェック
			if exists, err := gormutil.RecordExists(tx, &model.Stamp{Name: args.Name.String}); err != nil {
				return err
			} else if exists {
				return repository.ErrAlreadyExists
			}
			changes["name"] = args.Name.String
		}
		if args.FileID.Valid {
			// 存在チェック
			if args.FileID.UUID == uuid.Nil {
				return repository.ArgError("args.FileID", "FileID's file is not found")
			}
			if exists, err := gormutil.RecordExists(tx, &model.FileMeta{ID: args.FileID.UUID}); err != nil {
				return err
			} else if !exists {
				return repository.ArgError("args.FileID", "FileID's file is not found")
			}
			changes["file_id"] = args.FileID.UUID
		}
		if args.CreatorID.Valid {
			// uuid.Nilを許容する
			changes["creator_id"] = args.CreatorID.UUID
		}

		if len(changes) > 0 {
			return tx.Model(&s).Updates(changes).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		if repo.stamps != nil {
			repo.stamps.update(&s)
		}
		repo.hub.Publish(hub.Message{
			Name: event.StampUpdated,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
	}
	return nil
}

// GetStamp implements StampRepository interface.
func (repo *Repository) GetStamp(id uuid.UUID) (s *model.Stamp, err error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}

	if repo.stamps != nil {
		if s, ok := repo.stamps.GetStamp(id); ok {
			return s, nil
		}
		return nil, repository.ErrNotFound
	}

	s = &model.Stamp{}
	if err := repo.db.First(s, &model.Stamp{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return s, nil
}

// GetStampByName implements StampRepository interface.
func (repo *Repository) GetStampByName(name string) (s *model.Stamp, err error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	s = &model.Stamp{}
	if err := repo.db.First(s, &model.Stamp{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return s, nil
}

// DeleteStamp implements StampRepository interface.
func (repo *Repository) DeleteStamp(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	if repo.stamps != nil {
		repo.stamps.Lock()
		defer repo.stamps.Unlock()
	}

	result := repo.db.Delete(&model.Stamp{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		if repo.stamps != nil {
			repo.stamps.delete(id)
		}
		repo.hub.Publish(hub.Message{
			Name: event.StampDeleted,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
		return nil
	}
	return repository.ErrNotFound
}

// GetAllStamps implements StampRepository interface.
func (repo *Repository) GetAllStamps(stampType int) (stamps []*model.Stamp, err error) {
	stamps = make([]*model.Stamp, 0)
	tx := repo.db
	switch stampType {
	case 1:
		tx = tx.Where("is_unicode = TRUE")
	case 2:
		tx = tx.Where("is_unicode = FALSE")
	}
	return stamps, tx.Find(&stamps).Error
}

// GetStampsJSON implements StampRepository interface.
func (repo *Repository) GetStampsJSON(stampType int) ([]byte, time.Time, error) {
	if repo.stamps != nil {
		repo.stamps.RLock()
		defer repo.stamps.RUnlock()
		switch stampType {
		case 1:
			return repo.stamps.unicodeJSON, repo.stamps.updatedAt, nil
		case 2:
			return repo.stamps.originalJSON, repo.stamps.updatedAt, nil
		default:
			return repo.stamps.allJSON, repo.stamps.updatedAt, nil
		}
	}

	stamps, err := repo.GetAllStamps(stampType)
	if err != nil {
		return nil, time.Time{}, err
	}
	b, err := jsoniter.ConfigFastest.Marshal(stamps)
	return b, time.Now(), err
}

// StampExists implements StampRepository interface.
func (repo *Repository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}

	if repo.stamps != nil {
		_, ok := repo.stamps.GetStamp(id)
		return ok, nil
	}
	return gormutil.RecordExists(repo.db, &model.Stamp{ID: id})
}

// ExistStamps implements StampPaletteRepository interface.
func (repo *Repository) ExistStamps(stampIDs []uuid.UUID) (err error) {
	if repo.stamps != nil {
		if repo.stamps.CheckIDs(stampIDs) {
			return nil
		}
		return repository.ArgError("stamp", "stamp is not found")
	}

	num, err := gormutil.Count(repo.db.
		Table("stamps").
		Where("id IN (?)", stampIDs))
	if err != nil {
		return err
	}
	if len(stampIDs) != int(num) {
		err = repository.ArgError("stamp", "stamp is not found")
	}
	return
}

// GetUserStampHistory implements StampRepository interface.
func (repo *Repository) GetUserStampHistory(userID uuid.UUID, limit int) (h []*repository.UserStampHistory, err error) {
	h = make([]*repository.UserStampHistory, 0)
	if userID == uuid.Nil {
		return
	}

	err = repo.db.
		Table("messages_stamps ms1").
		Select("ms1.stamp_id, ms1.updated_at AS datetime").
		Joins("LEFT JOIN messages_stamps ms2 ON (ms1.updated_at < ms2.updated_at AND ms1.stamp_id = ms2.stamp_id AND ms1.user_id = ms2.user_id)").
		Where("ms2.stamp_id IS NULL AND ms1.user_id = ?", userID).
		Order("datetime DESC").
		Limit(limit).
		Scan(&h).
		Error
	return
}

// GetStampStats implements StampRepository interface
func (repo *Repository) GetStampStats(stampID uuid.UUID) (*repository.StampStats, error) {
	if stampID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	if ok, err := gormutil.
		RecordExists(repo.db, &model.MessageStamp{StampID: stampID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, repository.ErrNotFound
	}
	var stats repository.StampStats
	if err := repo.db.
		Unscoped().
		Model(&model.MessageStamp{}).
		Select("COUNT(stamp_id) AS count", "SUM(count) AS total_count").
		Where(&model.MessageStamp{StampID: stampID}).
		Find(&stats).
		Error; err != nil {
		return nil, err
	}
	return &stats, nil
}
