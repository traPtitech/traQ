package repository

import (
	"encoding/hex"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	timeoutDuration = 5 * time.Second
	tickTime        = 500 * time.Millisecond
)

type heartbeatImpl struct {
	hub                  *hub.Hub
	Ticker               *time.Ticker
	CurrentUserOnlineMap sync.Map
	HeartbeatStatuses    map[uuid.UUID]*model.HeartbeatStatus
	sync.RWMutex
}

func newHeartbeatImpl(hub *hub.Hub) *heartbeatImpl {
	h := &heartbeatImpl{
		hub:               hub,
		Ticker:            time.NewTicker(tickTime),
		HeartbeatStatuses: make(map[uuid.UUID]*model.HeartbeatStatus),
	}
	go func() {
		for {
			<-h.Ticker.C
			h.onTick()
		}
	}()
	return h
}

func (h *heartbeatImpl) onTick() {
	h.Lock()
	defer h.Unlock()
	timeout := time.Now().Add(-1 * timeoutDuration)
	updated := make(map[uuid.UUID]*model.HeartbeatStatus)
	for cid, channelStatus := range h.HeartbeatStatuses {
		arr := make([]*model.UserStatus, 0)
		for _, userStatus := range channelStatus.UserStatuses {
			if timeout.Before(userStatus.LastTime) {
				arr = append(arr, userStatus)
			} else {
				// 最終POSTから指定時間以上経ったものを削除する
				s, ok := h.CurrentUserOnlineMap.Load(userStatus.UserID)
				if ok {
					if toOffline := s.(*userOnlineStatus).dec(); toOffline {
						onlineUsersCounter.Dec()
						h.hub.Publish(hub.Message{
							Name: event.UserOffline,
							Fields: hub.Fields{
								"user_id":  userStatus.UserID,
								"datetime": s.(*userOnlineStatus).getTime(),
							},
						})
					}
				}
			}
		}
		if len(arr) > 0 {
			channelStatus.UserStatuses = arr
			updated[cid] = channelStatus
		}
	}
	h.HeartbeatStatuses = updated
}

type userOnlineStatus struct {
	sync.RWMutex
	id      uuid.UUID
	counter int
	time    time.Time
}

func (s *userOnlineStatus) isOnline() (r bool) {
	s.RLock()
	r = s.counter > 0
	s.RUnlock()
	return
}

func (s *userOnlineStatus) inc() (toOnline bool) {
	s.Lock()
	s.counter++
	if s.counter == 1 {
		toOnline = true
	}
	s.Unlock()
	return
}

func (s *userOnlineStatus) dec() (toOffline bool) {
	s.Lock()
	if s.counter > 0 {
		s.counter--
		if s.counter == 0 {
			toOffline = true
		}
	}
	s.Unlock()
	return
}

func (s *userOnlineStatus) setTime(time time.Time) {
	s.Lock()
	s.time = time
	s.Unlock()
}

func (s *userOnlineStatus) getTime() (t time.Time) {
	s.RLock()
	t = s.time
	s.RUnlock()
	return
}

// CreateUser implements UserRepository interface.
func (repo *GormRepository) CreateUser(name, password, role string) (*model.User, error) {
	salt := utils.GenerateSalt()
	user := &model.User{
		ID:       uuid.Must(uuid.NewV4()),
		Name:     name,
		Password: hex.EncodeToString(utils.HashPassword(password, salt)),
		Salt:     hex.EncodeToString(salt),
		Status:   model.UserAccountStatusActive,
		Bot:      false,
		Role:     role,
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	iconID, err := repo.GenerateIconFile(user.Name)
	if err != nil {
		return nil, err
	}
	user.Icon = iconID

	if err := repo.db.Create(user).Error; err != nil {
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
func (repo *GormRepository) GetUser(id uuid.UUID) (*model.User, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	return getUser(repo.db, &model.User{ID: id})
}

// GetUserByName implements UserRepository interface.
func (repo *GormRepository) GetUserByName(name string) (*model.User, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	return getUser(repo.db, &model.User{Name: name})
}

func getUser(tx *gorm.DB, where interface{}) (*model.User, error) {
	var user model.User
	if err := tx.First(&user, where).Error; err != nil {
		return nil, convertError(err)
	}
	return &user, nil
}

// GetUsers implements UserRepository interface.
func (repo *GormRepository) GetUsers() (users []*model.User, err error) {
	users = make([]*model.User, 0)
	err = repo.db.Find(&users).Error
	return users, err
}

// GetUserIDs implements UserRepository interface.
func (repo *GormRepository) GetUserIDs(query UsersQuery) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	tx := repo.db.Table("users")

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
	if query.IsSubscriberOf.Valid {
		tx = tx.Joins("INNER JOIN users_subscribe_channels ON users_subscribe_channels.user_id = users.id AND users_subscribe_channels.channel_id = ?", query.IsSubscriberOf.UUID)
	}
	if query.IsCMemberOf.Valid {
		tx = tx.Joins("INNER JOIN users_private_channels ON users_private_channels.user_id = users.id AND users_private_channels.channel_id = ?", query.IsCMemberOf.UUID)
	}
	if query.IsGMemberOf.Valid {
		tx = tx.Joins("INNER JOIN user_group_members ON user_group_members.user_id = users.id AND user_group_members.group_id = ?", query.IsGMemberOf.UUID)
	}

	err = tx.Pluck("users.id", &ids).Error
	return ids, err
}

// UserExists implements UserRepository interface.
func (repo *GormRepository) UserExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	return dbExists(repo.db, &model.User{ID: id})
}

// UpdateUser implements UserRepository interface.
func (repo *GormRepository) UpdateUser(id uuid.UUID, args UpdateUserArgs) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		u       model.User
		changed bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&u, model.User{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.DisplayName.Valid {
			if utf8.RuneCountInString(args.DisplayName.String) > 64 {
				return ArgError("args.DisplayName", "DisplayName must be shorter than 64 characters")
			}
			changes["display_name"] = args.DisplayName.String
		}
		if args.TwitterID.Valid {
			if len(args.TwitterID.String) > 0 && !validator.TwitterIDRegex.MatchString(args.TwitterID.String) {
				return ArgError("args.TwitterID", "invalid TwitterID")
			}
			changes["twitter_id"] = args.TwitterID.String
		}
		if args.Role.Valid {
			changes["role"] = args.Role.String
		}

		if len(changes) > 0 {
			if err := tx.Model(&u).Updates(changes).Error; err != nil {
				return err
			}
			changed = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.UserUpdated,
			Fields: hub.Fields{
				"user_id": id,
			},
		})
	}
	return nil
}

// ChangeUserPassword implements UserRepository interface.
func (repo *GormRepository) ChangeUserPassword(id uuid.UUID, password string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	if !validator.PasswordRegex.MatchString(password) {
		return ArgError("password", "invalid password characters")
	}
	salt := utils.GenerateSalt()
	return repo.db.Model(&model.User{ID: id}).Updates(map[string]interface{}{
		"salt":     hex.EncodeToString(salt),
		"password": hex.EncodeToString(utils.HashPassword(password, salt)),
	}).Error
}

// ChangeUserIcon implements UserRepository interface.
func (repo *GormRepository) ChangeUserIcon(id, fileID uuid.UUID) error {
	if id == uuid.Nil || fileID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Model(&model.User{ID: id}).Update("icon", fileID)
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserIconUpdated,
			Fields: hub.Fields{
				"user_id": id,
				"file_id": fileID,
			},
		})
	}
	return nil
}

// ChangeUserAccountStatus implements UserRepository interface.
func (repo *GormRepository) ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	if !status.Valid() {
		return ArgError("status", "invalid status")
	}
	result := repo.db.Model(&model.User{ID: id}).Update("status", status)
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserAccountStatusUpdated,
		Fields: hub.Fields{
			"user_id": id,
			"status":  status,
		},
	})
	return nil
}

// UpdateUserLastOnline implements UserRepository interface.
func (repo *GormRepository) UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error) {
	if id == uuid.Nil {
		return ErrNilID
	}
	return repo.db.Model(&model.User{ID: id}).Update("last_online", null.TimeFrom(time)).Error
}

// GetUserLastOnline implements UserRepository interface.
func (repo *GormRepository) GetUserLastOnline(id uuid.UUID) (time.Time, error) {
	if id == uuid.Nil {
		return time.Time{}, ErrNotFound
	}
	i, ok := repo.CurrentUserOnlineMap.Load(id)
	if !ok {
		var u model.User
		if err := repo.db.Select("last_online").First(&u, &model.User{ID: id}).Error; err != nil {
			return time.Time{}, convertError(err)
		}
		return u.LastOnline.Time, nil
	}
	return i.(*userOnlineStatus).getTime(), nil
}

// GetHeartbeatStatus implements UserRepository interface.
func (repo *GormRepository) GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool) {
	repo.heartbeatImpl.RLock()
	defer repo.heartbeatImpl.RUnlock()
	status, ok := repo.HeartbeatStatuses[channelID]
	if ok {
		return *status, ok
	}
	return model.HeartbeatStatus{}, ok
}

// UpdateHeartbeatStatus implements UserRepository interface.
func (repo *GormRepository) UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string) {
	repo.heartbeatImpl.Lock()
	defer repo.heartbeatImpl.Unlock()
	channelStatus, ok := repo.HeartbeatStatuses[channelID]
	if !ok {
		channelStatus = &model.HeartbeatStatus{ChannelID: channelID}
		repo.HeartbeatStatuses[channelID] = channelStatus
	}

	t := time.Now()
	s, _ := repo.CurrentUserOnlineMap.LoadOrStore(userID, &userOnlineStatus{id: userID})
	s.(*userOnlineStatus).setTime(t)
	for _, userStatus := range channelStatus.UserStatuses {
		if userStatus.UserID == userID {
			userStatus.LastTime = t
			userStatus.Status = status
			return
		}
	}
	userStatus := &model.UserStatus{
		UserID:   userID,
		Status:   status,
		LastTime: t,
	}
	channelStatus.UserStatuses = append(channelStatus.UserStatuses, userStatus)
	if toOnline := s.(*userOnlineStatus).inc(); toOnline {
		onlineUsersCounter.Inc()
		repo.hub.Publish(hub.Message{
			Name: event.UserOnline,
			Fields: hub.Fields{
				"user_id":  userID,
				"datetime": t,
			},
		})
	}
}

// IsUserOnline implements UserRepository interface.
func (repo *GormRepository) IsUserOnline(id uuid.UUID) bool {
	s, ok := repo.CurrentUserOnlineMap.Load(id)
	if !ok {
		return false
	}
	return s.(*userOnlineStatus).isOnline()
}
