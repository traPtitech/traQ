package repository

import (
	"encoding/hex"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
)

// CreateUser implements UserRepository interface.
func (repo *GormRepository) CreateUser(args CreateUserArgs) (model.UserInfo, error) {
	uid := uuid.Must(uuid.NewV4())
	user := &model.User{
		ID:          uid,
		Name:        args.Name,
		DisplayName: args.DisplayName,
		Icon:        args.IconFileID,
		Status:      model.UserAccountStatusActive,
		Bot:         false,
		Role:        args.Role,
		Profile:     &model.UserProfile{UserID: uid},
	}

	if len(args.Password) > 0 {
		salt := random.Salt()
		user.Password = hex.EncodeToString(utils.HashPassword(args.Password, salt))
		user.Salt = hex.EncodeToString(salt)
	}

	if args.ExternalLogin != nil {
		args.ExternalLogin.UserID = uid
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if exist, err := gormutil.RecordExists(tx, &model.User{Name: user.Name}); err != nil {
			return err
		} else if exist {
			return ErrAlreadyExists
		}

		// Create user, user_profile
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if args.ExternalLogin != nil {
			if err := tx.Create(args.ExternalLogin).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserCreated,
		Fields: hub.Fields{
			"user_id": user.ID,
			"user":    user,
		},
	})
	return user, nil
}

// GetUser implements UserRepository interface.
func (repo *GormRepository) GetUser(id uuid.UUID, withProfile bool) (model.UserInfo, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	return getUser(repo.db, withProfile, &model.User{ID: id})
}

// GetUserByName implements UserRepository interface.
func (repo *GormRepository) GetUserByName(name string, withProfile bool) (model.UserInfo, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	return getUser(repo.db, withProfile, &model.User{Name: name})
}

// GetUserByExternalID implements UserRepository interface.
func (repo *GormRepository) GetUserByExternalID(providerName, externalID string, withProfile bool) (model.UserInfo, error) {
	if len(providerName) == 0 || len(externalID) == 0 {
		return nil, ErrNotFound
	}
	var extUser model.ExternalProviderUser
	if err := repo.db.First(&extUser, &model.ExternalProviderUser{ProviderName: providerName, ExternalID: externalID}).Error; err != nil {
		return nil, convertError(err)
	}
	return getUser(repo.db, withProfile, &model.User{ID: extUser.UserID})
}

func getUser(tx *gorm.DB, withProfile bool, where ...interface{}) (model.UserInfo, error) {
	var user model.User
	if withProfile {
		tx = tx.Preload("Profile")
	}
	if err := tx.First(&user, where...).Error; err != nil {
		return nil, convertError(err)
	}
	return &user, nil
}

// GetUsers implements UserRepository interface.
func (repo *GormRepository) GetUsers(query UsersQuery) (users []model.UserInfo, err error) {
	arr := make([]*model.User, 0)
	if err = repo.makeGetUsersTx(query).Find(&arr).Error; err != nil {
		return nil, err
	}

	users = make([]model.UserInfo, len(arr))
	for i, u := range arr {
		users[i] = u
	}
	return users, nil
}

// GetUserIDs implements UserRepository interface.
func (repo *GormRepository) GetUserIDs(query UsersQuery) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	err = repo.makeGetUsersTx(query).Pluck("users.id", &ids).Error
	return ids, err
}

func (repo *GormRepository) makeGetUsersTx(query UsersQuery) *gorm.DB {
	tx := repo.db.Table("users")

	if query.Name.Valid {
		tx = tx.Where("users.name = ?", query.Name.String)
	}
	if query.IsActive.Valid {
		if query.IsActive.Bool {
			tx = tx.Where("users.status = ?", model.UserAccountStatusActive)
		} else {
			tx = tx.Where("users.status != ?", model.UserAccountStatusActive)
		}
	}
	if query.IsBot.Valid {
		tx = tx.Where("users.bot = ?", query.IsBot.Bool)
	}
	if query.IsSubscriberAtMarkLevelOf.Valid {
		tx = tx.Joins("INNER JOIN users_subscribe_channels ON users_subscribe_channels.user_id = users.id AND users_subscribe_channels.channel_id = ? AND users_subscribe_channels.mark = true", query.IsSubscriberAtMarkLevelOf.UUID)
	}
	if query.IsSubscriberAtNotifyLevelOf.Valid {
		tx = tx.Joins("INNER JOIN users_subscribe_channels ON users_subscribe_channels.user_id = users.id AND users_subscribe_channels.channel_id = ? AND users_subscribe_channels.notify = true", query.IsSubscriberAtNotifyLevelOf.UUID)
	}
	if query.IsCMemberOf.Valid {
		tx = tx.Joins("INNER JOIN users_private_channels ON users_private_channels.user_id = users.id AND users_private_channels.channel_id = ?", query.IsCMemberOf.UUID)
	}
	if query.IsGMemberOf.Valid {
		tx = tx.Joins("INNER JOIN user_group_members ON user_group_members.user_id = users.id AND user_group_members.group_id = ?", query.IsGMemberOf.UUID)
	}
	if query.EnableProfileLoading {
		tx = tx.Preload("Profile")
	}

	return tx
}

// UserExists implements UserRepository interface.
func (repo *GormRepository) UserExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	return gormutil.RecordExists(repo.db, &model.User{ID: id})
}

// UpdateUser implements UserRepository interface.
func (repo *GormRepository) UpdateUser(id uuid.UUID, args UpdateUserArgs) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		changed bool
		count   int
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var u model.User
		if err := tx.Preload("Profile").First(&u, model.User{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.DisplayName.Valid {
			if utf8.RuneCountInString(args.DisplayName.String) > 64 {
				return ArgError("args.DisplayName", "DisplayName must be shorter than 64 characters")
			}
			changes["display_name"] = args.DisplayName.String
		}
		if args.Role.Valid {
			changes["role"] = args.Role.String
		}
		if args.UserState.Valid {
			changes["status"] = args.UserState.State.Int()
		}
		if args.IconFileID.Valid {
			changes["icon"] = args.IconFileID.UUID
		}
		if args.Password.Valid {
			salt := random.Salt()
			changes["salt"] = hex.EncodeToString(salt)
			changes["password"] = hex.EncodeToString(utils.HashPassword(args.Password.String, salt))
		}
		if len(changes) > 0 {
			if err := tx.Model(&u).Updates(changes).Error; err != nil {
				return err
			}
			changed = true
			count += len(changes)
		}

		changes = map[string]interface{}{}
		if args.TwitterID.Valid {
			if len(args.TwitterID.String) > 0 && !validator.TwitterIDRegex.MatchString(args.TwitterID.String) {
				return ArgError("args.TwitterID", "invalid TwitterID")
			}
			changes["twitter_id"] = args.TwitterID.String
		}
		if args.Bio.Valid {
			changes["bio"] = args.Bio.String
		}
		if args.LastOnline.Valid {
			changes["last_online"] = args.LastOnline
		}
		if args.HomeChannel.Valid {
			if args.HomeChannel.UUID == uuid.Nil {
				changes["home_channel"] = optional.UUID{}
			} else {
				changes["home_channel"] = args.HomeChannel.UUID
			}
		}
		if len(changes) > 0 {
			if err := tx.Model(u.Profile).Updates(changes).Error; err != nil {
				return err
			}
			changed = true
			count += len(changes)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if args.Password.Valid && count == 2 {
		return nil // パスワードのみの変更の時はUserUpdatedイベントを発生させない
	}
	if args.LastOnline.Valid && count == 1 {
		return nil // 最終オンライン日時のみの更新の時はUserUpdatedイベントを発生させない
	}
	if changed {
		if args.IconFileID.Valid && count == 1 {
			repo.hub.Publish(hub.Message{
				Name: event.UserIconUpdated,
				Fields: hub.Fields{
					"user_id": id,
					"file_id": args.IconFileID.UUID,
				},
			})
		} else {
			repo.hub.Publish(hub.Message{
				Name: event.UserUpdated,
				Fields: hub.Fields{
					"user_id": id,
				},
			})
		}
	}
	return nil
}

// LinkExternalUserAccount implements UserRepository interface.
func (repo *GormRepository) LinkExternalUserAccount(userID uuid.UUID, args LinkExternalUserAccountArgs) error {
	if userID == uuid.Nil {
		return ErrNilID
	}
	if len(args.ProviderName) == 0 {
		return ArgError("args.ProviderName", "ProviderName must not be empty")
	}
	if len(args.ExternalID) == 0 {
		return ArgError("args.ExternalID", "ExternalID must not be empty")
	}

	return repo.db.Transaction(func(tx *gorm.DB) error {
		if exist, err := gormutil.RecordExists(tx, &model.User{ID: userID}); err != nil {
			return err
		} else if !exist {
			return ErrNotFound
		}

		if exist, err := gormutil.RecordExists(tx, &model.ExternalProviderUser{UserID: userID, ProviderName: args.ProviderName}); err != nil {
			return err
		} else if exist {
			return ErrAlreadyExists
		}

		if exist, err := gormutil.RecordExists(tx, &model.ExternalProviderUser{ProviderName: args.ProviderName, ExternalID: args.ExternalID}); err != nil {
			return err
		} else if exist {
			return ErrAlreadyExists
		}

		return tx.Create(&model.ExternalProviderUser{
			UserID:       userID,
			ProviderName: args.ProviderName,
			ExternalID:   args.ExternalID,
			Extra:        args.Extra,
		}).Error
	})
}

// GetLinkedExternalUserAccounts implements UserRepository interface.
func (repo *GormRepository) GetLinkedExternalUserAccounts(userID uuid.UUID) ([]*model.ExternalProviderUser, error) {
	result := make([]*model.ExternalProviderUser, 0)
	if userID == uuid.Nil {
		return result, nil
	}
	return result, repo.db.Find(&result, &model.ExternalProviderUser{UserID: userID}).Error
}

// UnlinkExternalUserAccount implements UserRepository interface.
func (repo *GormRepository) UnlinkExternalUserAccount(userID uuid.UUID, providerName string) error {
	if userID == uuid.Nil || len(providerName) == 0 {
		return ErrNilID
	}

	result := repo.db.Delete(model.ExternalProviderUser{}, &model.ExternalProviderUser{UserID: userID, ProviderName: providerName})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetUserStats implements UserRepository interface
func (repo *GormRepository) GetUserStats(userID uuid.UUID) (*UserStats, error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	if ok, err := gormutil.
		RecordExists(repo.db, &model.User{ID: userID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotFound
	}
	var stats UserStats
	if err := repo.db.
		Unscoped().
		Model(&model.Message{}).
		Where(&model.Message{UserID: userID}).
		Count(&stats.TotalMessageCount).
		Error; err != nil {
		return nil, err
	}
	var allStampCount []struct {
		StampID    uuid.UUID
		Count      int64
		TotalCount int64
	}

	if err := repo.db.
		Unscoped().
		Model(&model.MessageStamp{}).
		Select("stamp_id AS stamp_id", "COUNT(stamp_id) AS count", "SUM(count) AS total_count").
		Where(&model.MessageStamp{UserID: userID}).
		Group("stamp_id").
		Find(&allStampCount).
		Error; err != nil {
		return nil, err
	}
	stats.StampCount = make(map[uuid.UUID]int64)
	stats.TotalStampCount = make(map[uuid.UUID]int64)

	for _, stampCount := range allStampCount {
		stats.StampCount[stampCount.StampID] = stampCount.Count
		stats.TotalStampCount[stampCount.StampID] = stampCount.TotalCount
	}

	stats.DateTime = time.Now()
	return &stats, nil
}
