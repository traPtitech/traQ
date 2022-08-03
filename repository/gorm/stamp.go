package gorm

import (
	"context"
	"errors"
	"sort"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/motoki317/sc"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormUtil"
	"github.com/traPtitech/traQ/utils/validator"
)

type stampRepository struct {
	stamps  *sc.Cache[struct{}, map[uuid.UUID]*model.Stamp]
	perType *sc.Cache[repository.StampType, []*model.Stamp]
}

func makeStampRepository(db *gorm.DB) *stampRepository {
	// Lazy load
	r := &stampRepository{}
	r.stamps = sc.NewMust(r.loadFunc(db), 365*24*time.Hour, 365*24*time.Hour)
	r.perType = sc.NewMust(r.filterFunc(), 365*24*time.Hour, 365*24*time.Hour)
	return r
}

func (r *stampRepository) loadFunc(db *gorm.DB) func(context.Context, struct{}) (map[uuid.UUID]*model.Stamp, error) {
	return func(_ context.Context, _ struct{}) (map[uuid.UUID]*model.Stamp, error) {
		var stamps []*model.Stamp
		if err := db.Find(&stamps).Error; err != nil {
			return nil, err
		}
		stampsMap := make(map[uuid.UUID]*model.Stamp, len(stamps))
		for _, s := range stamps {
			stampsMap[s.ID] = s
		}
		return stampsMap, nil
	}
}

func (r *stampRepository) filterFunc() func(_ context.Context, stampType repository.StampType) ([]*model.Stamp, error) {
	return func(ctx context.Context, stampType repository.StampType) ([]*model.Stamp, error) {
		stamps, err := r.stamps.Get(ctx, struct{}{})
		if err != nil {
			return nil, err
		}
		arr := make([]*model.Stamp, 0, len(stamps))

		switch stampType {
		case repository.StampTypeAll:
			for _, s := range stamps {
				arr = append(arr, s)
			}
		case repository.StampTypeUnicode:
			for _, s := range stamps {
				if s.IsUnicode {
					arr = append(arr, s)
				}
			}
		case repository.StampTypeOriginal:
			for _, s := range stamps {
				if !s.IsUnicode {
					arr = append(arr, s)
				}
			}
		default:
			return nil, errors.New("unknown stamp type")
		}

		sort.Slice(arr, func(i, j int) bool { return arr[i].ID.String() < arr[j].ID.String() })
		return arr, nil
	}
}

// Purge purges stamp cache.
func (r *stampRepository) Purge() {
	r.stamps.Purge()
	r.perType.Purge()
}

func (r *stampRepository) GetStamp(id uuid.UUID) (s *model.Stamp, ok bool, err error) {
	stamps, err := r.stamps.Get(context.Background(), struct{}{})
	if err != nil {
		return nil, false, err
	}
	s, ok = stamps[id]
	return
}

func (r *stampRepository) CheckIDs(ids []uuid.UUID) (ok bool, err error) {
	stamps, err := r.stamps.Get(context.Background(), struct{}{})
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if _, ok := stamps[id]; !ok {
			return false, nil
		}
	}
	return true, nil
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

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if err := vd.Validate(stamp.Name, validator.StampNameRuleRequired...); err != nil {
			return repository.ArgError("name", "Name must be 1-32 characters of a-zA-Z0-9_-")
		}
		// 名前重複チェック
		if exists, err := gormUtil.RecordExists(tx, &model.Stamp{Name: stamp.Name}); err != nil {
			return err
		} else if exists {
			return repository.ErrAlreadyExists
		}
		// ファイル存在チェック
		if stamp.FileID == uuid.Nil {
			return repository.ArgError("fileID", "FileID's file is not found")
		}
		if exists, err := gormUtil.RecordExists(tx, &model.FileMeta{ID: stamp.FileID}); err != nil {
			return err
		} else if !exists {
			return repository.ArgError("fileID", "fileID's file is not found")
		}

		return tx.Create(stamp).Error
	})
	if err != nil {
		return nil, err
	}

	repo.stamps.Purge()

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
			if exists, err := gormUtil.RecordExists(tx, &model.Stamp{Name: args.Name.String}); err != nil {
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
			if exists, err := gormUtil.RecordExists(tx, &model.FileMeta{ID: args.FileID.UUID}); err != nil {
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
		repo.stamps.Purge()
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

	s, ok, err := repo.stamps.GetStamp(id)
	if err != nil {
		return nil, err
	}
	if ok {
		return s, nil
	}
	return nil, repository.ErrNotFound
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

	result := repo.db.Delete(&model.Stamp{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.stamps.Purge()
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
func (repo *Repository) GetAllStamps(stampType repository.StampType) (stamps []*model.Stamp, err error) {
	return repo.stamps.perType.Get(context.Background(), stampType)
}

// StampExists implements StampRepository interface.
func (repo *Repository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}

	_, ok, err := repo.stamps.GetStamp(id)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ExistStamps implements StampPaletteRepository interface.
func (repo *Repository) ExistStamps(stampIDs []uuid.UUID) (err error) {
	ok, err := repo.stamps.CheckIDs(stampIDs)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return repository.ArgError("stamp", "stamp is not found")
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

	if ok, err := gormUtil.
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
