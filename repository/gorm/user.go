package gorm

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/motoki317/sc"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
)

var _ repository.UserRepository = (*userRepository)(nil)

type getUserArg struct {
	id          uuid.UUID
	withProfile bool
}

type userRepository struct {
	db    *gorm.DB
	hub   *hub.Hub
	users *sc.Cache[getUserArg, model.UserInfo]
}

func makeUserRepository(db *gorm.DB, hub *hub.Hub) *userRepository {
	r := &userRepository{db: db, hub: hub}
	r.users = sc.NewMust(r.getUser, 1*time.Hour, 1*time.Hour)
	return r
}

func (r *userRepository) getUser(_ context.Context, arg getUserArg) (model.UserInfo, error) {
	tx := r.db
	if arg.withProfile {
		tx = tx.Preload("Profile")
	}
	var user model.User
	if err := tx.First(&user, &model.User{ID: arg.id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &user, nil
}

func (r *userRepository) forgetCache(id uuid.UUID) {
	r.users.Forget(getUserArg{id, false})
	r.users.Forget(getUserArg{id, true})
}

// CreateUser implements UserRepository interface.
func (r *userRepository) CreateUser(args repository.CreateUserArgs) (model.UserInfo, error) {
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

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if exist, err := gormutil.RecordExists(tx, &model.User{Name: user.Name}); err != nil {
			return err
		} else if exist {
			return repository.ErrAlreadyExists
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
	r.hub.Publish(hub.Message{
		Name: event.UserCreated,
		Fields: hub.Fields{
			"user_id": user.ID,
			"user":    user,
		},
	})
	return user, nil
}

// GetUser implements UserRepository interface.
func (r *userRepository) GetUser(id uuid.UUID, withProfile bool) (model.UserInfo, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	return r.users.Get(context.Background(), getUserArg{id, withProfile})
}

// GetUserByName implements UserRepository interface.
func (r *userRepository) GetUserByName(name string, withProfile bool) (model.UserInfo, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}

	tx := r.db
	if withProfile {
		tx = tx.Preload("Profile")
	}
	var user model.User
	if err := tx.First(&user, &model.User{Name: name}).Error; err != nil {
		return nil, convertError(err)
	}
	return &user, nil
}

// GetUserByExternalID implements UserRepository interface.
func (r *userRepository) GetUserByExternalID(providerName, externalID string, withProfile bool) (model.UserInfo, error) {
	if len(providerName) == 0 || len(externalID) == 0 {
		return nil, repository.ErrNotFound
	}
	var extUser model.ExternalProviderUser
	if err := r.db.First(&extUser, &model.ExternalProviderUser{ProviderName: providerName, ExternalID: externalID}).Error; err != nil {
		return nil, convertError(err)
	}
	return r.users.Get(context.Background(), getUserArg{extUser.UserID, withProfile})
}

// GetUsers implements UserRepository interface.
func (r *userRepository) GetUsers(query repository.UsersQuery) (users []model.UserInfo, err error) {
	arr := make([]*model.User, 0)
	if err = r.makeGetUsersTx(query).Find(&arr).Error; err != nil {
		return nil, err
	}

	users = make([]model.UserInfo, len(arr))
	for i, u := range arr {
		users[i] = u
	}
	return users, nil
}

// GetUserIDs implements UserRepository interface.
func (r *userRepository) GetUserIDs(query repository.UsersQuery) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	err = r.makeGetUsersTx(query).Pluck("users.id", &ids).Error
	return ids, err
}

func (r *userRepository) makeGetUsersTx(query repository.UsersQuery) *gorm.DB {
	tx := r.db.Table("users")

	if query.Name.Valid {
		tx = tx.Where("users.name = ?", query.Name.V)
	}
	if query.IsActive.Valid {
		if query.IsActive.V {
			tx = tx.Where("users.status = ?", model.UserAccountStatusActive)
		} else {
			tx = tx.Where("users.status != ?", model.UserAccountStatusActive)
		}
	}
	if query.IsBot.Valid {
		tx = tx.Where("users.bot = ?", query.IsBot.V)
	}
	if query.IsSubscriberAtMarkLevelOf.Valid {
		tx = tx.Joins("INNER JOIN users_subscribe_channels ON users_subscribe_channels.user_id = users.id AND users_subscribe_channels.channel_id = ? AND users_subscribe_channels.mark = true", query.IsSubscriberAtMarkLevelOf.V)
	}
	if query.IsSubscriberAtNotifyLevelOf.Valid {
		tx = tx.Joins("INNER JOIN users_subscribe_channels ON users_subscribe_channels.user_id = users.id AND users_subscribe_channels.channel_id = ? AND users_subscribe_channels.notify = true", query.IsSubscriberAtNotifyLevelOf.V)
	}
	if query.IsCMemberOf.Valid {
		tx = tx.Joins("INNER JOIN users_private_channels ON users_private_channels.user_id = users.id AND users_private_channels.channel_id = ?", query.IsCMemberOf.V)
	}
	if query.IsGMemberOf.Valid {
		tx = tx.Joins("INNER JOIN user_group_members ON user_group_members.user_id = users.id AND user_group_members.group_id = ?", query.IsGMemberOf.V)
	}
	if query.EnableProfileLoading {
		tx = tx.Preload("Profile")
	}

	return tx
}

// UserExists implements UserRepository interface.
func (r *userRepository) UserExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	return gormutil.RecordExists(r.db, &model.User{ID: id})
}

// UpdateUser implements UserRepository interface.
func (r *userRepository) UpdateUser(id uuid.UUID, args repository.UpdateUserArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var u model.User

	var (
		activate   bool
		deactivate bool
		changed    bool
		count      int
	)
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Profile").First(&u, model.User{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.DisplayName.Valid {
			changes["display_name"] = args.DisplayName.V
		}
		if args.Role.Valid {
			changes["role"] = args.Role.V
		}
		if args.UserState.Valid {
			changes["status"] = args.UserState.V.Int()
			switch args.UserState.V.Int() {
			case model.UserAccountStatusActive.Int():
				activate = true
			case model.UserAccountStatusDeactivated.Int():
				deactivate = true
			}
		}
		if args.IconFileID.Valid {
			changes["icon"] = args.IconFileID.V
		}
		if args.Password.Valid {
			salt := random.Salt()
			changes["salt"] = hex.EncodeToString(salt)
			changes["password"] = hex.EncodeToString(utils.HashPassword(args.Password.V, salt))
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
			if len(args.TwitterID.V) > 0 && !validator.TwitterIDRegex.MatchString(args.TwitterID.V) {
				return repository.ArgError("args.TwitterID", "invalid TwitterID")
			}
			changes["twitter_id"] = args.TwitterID.V
		}
		if args.Bio.Valid {
			changes["bio"] = args.Bio.V
		}
		if args.LastOnline.Valid {
			changes["last_online"] = args.LastOnline
		}
		if args.HomeChannel.Valid {
			if args.HomeChannel.V == uuid.Nil {
				changes["home_channel"] = optional.Of[uuid.UUID]{}
			} else {
				changes["home_channel"] = args.HomeChannel.V
			}
		}
		if len(changes) > 0 {
			if err := tx.Model(u.Profile).Updates(changes).Error; err != nil {
				return err
			}
			changed = true
			count += len(changes)
		}
		// 凍結の際は未読を削除
		if deactivate {
			if err := tx.Delete(&model.Unread{}, "user_id = ?", id).Error; err != nil {
				return err
			}
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
	if !changed {
		return nil
	}

	r.forgetCache(id)
	switch {
	case args.IconFileID.Valid && count == 1:
		r.hub.Publish(hub.Message{
			Name: event.UserIconUpdated,
			Fields: hub.Fields{
				"user_id": id,
				"file_id": args.IconFileID.V,
			},
		})
	case activate:
		r.hub.Publish(hub.Message{
			Name: event.UserActivated,
			Fields: hub.Fields{
				"user": &u,
			},
		})
	default:
		r.hub.Publish(hub.Message{
			Name: event.UserUpdated,
			Fields: hub.Fields{
				"user_id": id,
			},
		})
	}
	return nil
}

// LinkExternalUserAccount implements UserRepository interface.
func (r *userRepository) LinkExternalUserAccount(userID uuid.UUID, args repository.LinkExternalUserAccountArgs) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}
	if len(args.ProviderName) == 0 {
		return repository.ArgError("args.ProviderName", "ProviderName must not be empty")
	}
	if len(args.ExternalID) == 0 {
		return repository.ArgError("args.ExternalID", "ExternalID must not be empty")
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if exist, err := gormutil.RecordExists(tx, &model.User{ID: userID}); err != nil {
			return err
		} else if !exist {
			return repository.ErrNotFound
		}

		if exist, err := gormutil.RecordExists(tx, &model.ExternalProviderUser{UserID: userID, ProviderName: args.ProviderName}); err != nil {
			return err
		} else if exist {
			return repository.ErrAlreadyExists
		}

		if exist, err := gormutil.RecordExists(tx, &model.ExternalProviderUser{ProviderName: args.ProviderName, ExternalID: args.ExternalID}); err != nil {
			return err
		} else if exist {
			return repository.ErrAlreadyExists
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
func (r *userRepository) GetLinkedExternalUserAccounts(userID uuid.UUID) ([]*model.ExternalProviderUser, error) {
	result := make([]*model.ExternalProviderUser, 0)
	if userID == uuid.Nil {
		return result, nil
	}
	return result, r.db.Find(&result, &model.ExternalProviderUser{UserID: userID}).Error
}

// UnlinkExternalUserAccount implements UserRepository interface.
func (r *userRepository) UnlinkExternalUserAccount(userID uuid.UUID, providerName string) error {
	if userID == uuid.Nil || len(providerName) == 0 {
		return repository.ErrNilID
	}

	result := r.db.Delete(model.ExternalProviderUser{}, &model.ExternalProviderUser{UserID: userID, ProviderName: providerName})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// GetUserStats implements UserRepository interface
func (r *userRepository) GetUserStats(userID uuid.UUID) (*repository.UserStats, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	if ok, err := gormutil.
		RecordExists(r.db, &model.User{ID: userID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, repository.ErrNotFound
	}
	var stats repository.UserStats
	if err := r.db.
		Unscoped().
		Model(&model.Message{}).
		Where(&model.Message{UserID: userID}).
		Count(&stats.TotalMessageCount).
		Error; err != nil {
		return nil, err
	}

	if err := r.db.
		Unscoped().
		Model(&model.MessageStamp{}).
		Select("stamp_id AS id", "COUNT(stamp_id) AS count", "SUM(count) AS total").
		Where(&model.MessageStamp{UserID: userID}).
		Group("stamp_id").
		Order("count DESC").
		Find(&stats.Stamps).
		Error; err != nil {
		return nil, err
	}

	stats.DateTime = time.Now()
	return &stats, nil
}
