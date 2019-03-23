package impl

import (
	"encoding/hex"
	"errors"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
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

// CreateUser ユーザーを作成します
func (repo *RepositoryImpl) CreateUser(name, password string, role gorbac.Role) (*model.User, error) {
	salt := utils.GenerateSalt()
	user := &model.User{
		ID:       uuid.Must(uuid.NewV4()),
		Name:     name,
		Password: hex.EncodeToString(utils.HashPassword(password, salt)),
		Salt:     hex.EncodeToString(salt),
		Status:   model.UserAccountStatusActive,
		Bot:      false,
		Role:     role.ID(),
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

// GetUser ユーザーを取得する
func (repo *RepositoryImpl) GetUser(id uuid.UUID) (*model.User, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	user := &model.User{}
	if err := repo.db.Where(&model.User{ID: id}).Take(user).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUserByName ユーザーを取得する
func (repo *RepositoryImpl) GetUserByName(name string) (*model.User, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	user := &model.User{}
	if err := repo.db.Where(&model.User{Name: name}).Take(user).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUsers 全ユーザーの取得
func (repo *RepositoryImpl) GetUsers() (users []*model.User, err error) {
	users = make([]*model.User, 0)
	err = repo.db.Find(&users).Error
	return users, err
}

// UserExists 指定したIDのユーザーが存在するかどうか
func (repo *RepositoryImpl) UserExists(id uuid.UUID) (bool, error) {
	c := 0
	err := repo.db.
		Model(&model.User{}).
		Where(&model.User{ID: id}).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}

// ChangeUserDisplayName ユーザーの表示名を変更します
func (repo *RepositoryImpl) ChangeUserDisplayName(id uuid.UUID, displayName string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if utf8.RuneCountInString(displayName) > 64 {
		return errors.New("displayName must be <=64 characters")
	}
	result := repo.db.Model(&model.User{ID: id}).Update("display_name", displayName)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserUpdated,
			Fields: hub.Fields{
				"user_id": id,
			},
		})
	}
	return nil
}

// ChangeUserPassword ユーザーのパスワードを変更します
func (repo *RepositoryImpl) ChangeUserPassword(id uuid.UUID, password string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	salt := utils.GenerateSalt()
	result := repo.db.Model(&model.User{ID: id}).Updates(map[string]interface{}{
		"salt":     hex.EncodeToString(salt),
		"password": hex.EncodeToString(utils.HashPassword(password, salt)),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.UserUpdated,
			Fields: hub.Fields{
				"user_id": id,
			},
		})
	}
	return nil
}

// ChangeUserIcon ユーザーのアイコンを変更します
func (repo *RepositoryImpl) ChangeUserIcon(id, fileID uuid.UUID) error {
	if id == uuid.Nil || fileID == uuid.Nil {
		return repository.ErrNilID
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

// ChangeUserTwitterID ユーザーのTwitterIDを変更します
func (repo *RepositoryImpl) ChangeUserTwitterID(id uuid.UUID, twitterID string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if err := validator.ValidateVar(twitterID, "twitterid"); err != nil {
		return err
	}
	return repo.db.Model(&model.User{ID: id}).Update("twitter_id", twitterID).Error
}

// ChangeUserAccountStatus ユーザーのアカウント状態を変更します
func (repo *RepositoryImpl) ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Model(&model.User{ID: id}).Update("status", status)
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
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

// UpdateUserLastOnline ユーザーの最終オンライン日時を更新します
func (repo *RepositoryImpl) UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	return repo.db.Model(&model.User{ID: id}).Update("last_online", &time).Error
}

// GetUserLastOnline ユーザーの最終オンライン日時を取得します
func (repo *RepositoryImpl) GetUserLastOnline(id uuid.UUID) (time.Time, error) {
	i, ok := repo.CurrentUserOnlineMap.Load(id)
	if !ok {
		var u model.User
		if err := repo.db.Model(&model.User{ID: id}).Select("last_online").Take(&u).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return time.Time{}, repository.ErrNotFound
			}
			return time.Time{}, err
		}
		if u.LastOnline == nil {
			return time.Time{}, nil
		}
		return *u.LastOnline, nil
	}
	return i.(*userOnlineStatus).getTime(), nil
}

// GetHeartbeatStatus channelIDで指定したHeartbeatStatusを取得する
func (repo *RepositoryImpl) GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool) {
	repo.heartbeatImpl.RLock()
	defer repo.heartbeatImpl.RUnlock()
	status, ok := repo.HeartbeatStatuses[channelID]
	if ok {
		return *status, ok
	}
	return model.HeartbeatStatus{}, ok
}

// UpdateHeartbeatStatus UserIDで指定されたUserのHeartbeatの更新を行う
func (repo *RepositoryImpl) UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string) {
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
		repo.hub.Publish(hub.Message{
			Name: event.UserOnline,
			Fields: hub.Fields{
				"user_id":  userID,
				"datetime": t,
			},
		})
	}
}

// IsUserOnline ユーザーがオンラインかどうかを返します。
func (repo *RepositoryImpl) IsUserOnline(id uuid.UUID) bool {
	s, ok := repo.CurrentUserOnlineMap.Load(id)
	if !ok {
		return false
	}
	return s.(*userOnlineStatus).isOnline()
}
