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
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/validator"
)

var _ repository.StampRepository = (*stampRepository)(nil)

type stampRepository struct {
	db      *gorm.DB
	hub     *hub.Hub
	stamps  *sc.Cache[struct{}, map[uuid.UUID]*model.Stamp]
	perType *sc.Cache[repository.StampType, []*model.StampWithThumbnail]
}

func makeStampRepository(db *gorm.DB, hub *hub.Hub) *stampRepository {
	// Lazy load
	r := &stampRepository{db: db, hub: hub}
	r.stamps = sc.NewMust(r.loadStamps, 365*24*time.Hour, 365*24*time.Hour)
	r.perType = sc.NewMust(r.loadFilteredStamps, 365*24*time.Hour, 365*24*time.Hour)
	return r
}

func (r *stampRepository) loadStamps(_ context.Context, _ struct{}) (map[uuid.UUID]*model.Stamp, error) {
	var stamps []*model.Stamp
	if err := r.db.Find(&stamps).Error; err != nil {
		return nil, err
	}
	stampsMap := make(map[uuid.UUID]*model.Stamp, len(stamps))
	for _, s := range stamps {
		stampsMap[s.ID] = s
	}
	return stampsMap, nil
}

func (r *stampRepository) loadFilteredStamps(ctx context.Context, stampType repository.StampType) ([]*model.StampWithThumbnail, error) {
	stamps, err := r.stamps.Get(ctx, struct{}{})
	if err != nil {
		return nil, err
	}
	arr := make([]*model.StampWithThumbnail, 0, len(stamps))

	IDs := make([]uuid.UUID, 0, len(stamps))
	stampsWithThumbnail := make([]*model.StampWithThumbnail, 0, len(stamps))
	for _, stamp := range stamps {
		IDs = append(IDs, stamp.FileID)
	}

	thumbnails := make([]uuid.UUID, 0, len(stamps))
	if err := r.db.
		Table("files_thumbnails").
		Select("file_id").
		Where("file_id IN (?)", IDs).
		Find(&thumbnails).
		Error; err != nil {
		return nil, err
	}
	thumbnailExists := make(map[uuid.UUID]struct{}, len(thumbnails))
	for _, v := range thumbnails {
		thumbnailExists[v] = struct{}{}
	}

	for _, stamp := range stamps {
		_, ok := thumbnailExists[stamp.FileID]
		stampsWithThumbnail = append(stampsWithThumbnail, &model.StampWithThumbnail{
			Stamp:        stamp,
			HasThumbnail: ok,
		})
	}

	if err != nil {
		return nil, err
	}
	switch stampType {
	case repository.StampTypeAll:
		arr = append(arr, stampsWithThumbnail...)
	case repository.StampTypeUnicode:
		for _, s := range stampsWithThumbnail {
			if s.IsUnicode {
				arr = append(arr, s)
			}
		}
	case repository.StampTypeOriginal:
		for _, s := range stampsWithThumbnail {
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

func (r *stampRepository) purgeCache() {
	r.stamps.Purge()
	r.perType.Purge()
}

func (r *stampRepository) getStamp(id uuid.UUID) (s *model.Stamp, ok bool, err error) {
	stamps, err := r.stamps.Get(context.Background(), struct{}{})
	if err != nil {
		return nil, false, err
	}
	s, ok = stamps[id]
	return
}

func (r *stampRepository) allStampsExist(ids []uuid.UUID) (ok bool, err error) {
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
func (r *stampRepository) CreateStamp(args repository.CreateStampArgs) (s *model.Stamp, err error) {
	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      args.Name,
		FileID:    args.FileID,
		CreatorID: args.CreatorID, // uuid.Nilを許容する
		IsUnicode: args.IsUnicode,
	}

	err = r.db.Transaction(func(tx *gorm.DB) error {
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

	r.purgeCache()

	r.hub.Publish(hub.Message{
		Name: event.StampCreated,
		Fields: hub.Fields{
			"stamp":    stamp,
			"stamp_id": stamp.ID,
		},
	})
	return stamp, nil
}

// UpdateStamp implements StampRepository interface.
func (r *stampRepository) UpdateStamp(id uuid.UUID, args repository.UpdateStampArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	var s model.Stamp
	changes := map[string]interface{}{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&s, &model.Stamp{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if args.Name.Valid && s.Name != args.Name.V {
			if err := vd.Validate(args.Name.V, validator.StampNameRuleRequired...); err != nil {
				return repository.ArgError("args.Name", "Name must be 1-32 characters of a-zA-Z0-9_-")
			}

			// 重複チェック
			if exists, err := gormutil.RecordExists(tx, &model.Stamp{Name: args.Name.V}); err != nil {
				return err
			} else if exists {
				return repository.ErrAlreadyExists
			}
			changes["name"] = args.Name.V
		}
		if args.FileID.Valid {
			// 存在チェック
			if args.FileID.V == uuid.Nil {
				return repository.ArgError("args.FileID", "FileID's file is not found")
			}
			if exists, err := gormutil.RecordExists(tx, &model.FileMeta{ID: args.FileID.V}); err != nil {
				return err
			} else if !exists {
				return repository.ArgError("args.FileID", "FileID's file is not found")
			}
			changes["file_id"] = args.FileID.V
		}
		if args.CreatorID.Valid {
			// uuid.Nilを許容する
			changes["creator_id"] = args.CreatorID.V
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
		r.purgeCache()
		r.hub.Publish(hub.Message{
			Name: event.StampUpdated,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
	}
	return nil
}

// GetStamp implements StampRepository interface.
func (r *stampRepository) GetStamp(id uuid.UUID) (s *model.Stamp, err error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}

	s, ok, err := r.getStamp(id)
	if err != nil {
		return nil, err
	}
	if ok {
		return s, nil
	}
	return nil, repository.ErrNotFound
}

// GetStampByName implements StampRepository interface.
func (r *stampRepository) GetStampByName(name string) (s *model.Stamp, err error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	s = &model.Stamp{}
	if err := r.db.First(s, &model.Stamp{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return s, nil
}

// DeleteStamp implements StampRepository interface.
func (r *stampRepository) DeleteStamp(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	result := r.db.Delete(&model.Stamp{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		r.purgeCache()
		r.hub.Publish(hub.Message{
			Name: event.StampDeleted,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
		return nil
	}
	return repository.ErrNotFound
}

// GetAllStampsWithThumbnail implements StampRepository interface.
func (r *stampRepository) GetAllStampsWithThumbnail(stampType repository.StampType) (stampsWithThumbnail []*model.StampWithThumbnail, err error) {
	return r.perType.Get(context.Background(), stampType)
}

// StampExists implements StampRepository interface.
func (r *stampRepository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}

	_, ok, err := r.getStamp(id)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ExistStamps implements StampPaletteRepository interface.
func (r *stampRepository) ExistStamps(stampIDs []uuid.UUID) (err error) {
	ok, err := r.allStampsExist(stampIDs)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return repository.ArgError("stamp", "stamp is not found")
}

// GetUserStampHistory implements StampRepository interface.
func (r *stampRepository) GetUserStampHistory(userID uuid.UUID, limit int) (h []*repository.UserStampHistory, err error) {
	h = make([]*repository.UserStampHistory, 0)
	if userID == uuid.Nil {
		return
	}

	err = r.db.
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
func (r *stampRepository) GetStampStats(stampID uuid.UUID) (*repository.StampStats, error) {
	if stampID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	if ok, err := gormutil.
		RecordExists(r.db, &model.MessageStamp{StampID: stampID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, repository.ErrNotFound
	}
	var stats repository.StampStats
	if err := r.db.
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
