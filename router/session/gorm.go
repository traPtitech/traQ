package session

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/random"
)

func init() {
	gob.Register(map[string]interface{}{})
}

type session struct {
	t         string
	refID     uuid.UUID
	userID    uuid.UUID
	createdAt time.Time

	loaded bool
	db     *gorm.DB
	data   map[string]interface{}
	sync.Mutex
}

func newSession(db *gorm.DB, t string, refID uuid.UUID, userID uuid.UUID, createdAt time.Time, data map[string]interface{}) *session {
	return &session{
		t:         t,
		refID:     refID,
		userID:    userID,
		createdAt: createdAt,
		loaded:    data != nil,
		db:        db,
		data:      data,
	}
}

func (s *session) Token() string {
	return s.t
}

func (s *session) RefID() uuid.UUID {
	return s.refID
}

func (s *session) UserID() uuid.UUID {
	return s.userID
}

func (s *session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *session) LoggedIn() bool {
	return s.userID != uuid.Nil
}

func (s *session) Get(key string) (interface{}, error) {
	s.Lock()
	defer s.Unlock()
	if !s.loaded {
		if err := s.load(); err != nil {
			return nil, err
		}
	}
	v := s.data[key]
	return v, nil
}

func (s *session) Set(key string, value interface{}) error {
	s.Lock()
	defer s.Unlock()
	if !s.loaded {
		if err := s.load(); err != nil {
			return err
		}
	}
	s.data[key] = value
	return s.save()
}

func (s *session) Delete(key string) error {
	s.Lock()
	defer s.Unlock()
	if !s.loaded {
		if err := s.load(); err != nil {
			return err
		}
	}
	delete(s.data, key)
	return s.save()
}

func (s *session) Expired() bool {
	return time.Since(s.createdAt) > time.Duration(sessionMaxAge)*time.Second
}

func (s *session) Refreshable() bool {
	return time.Since(s.createdAt) <= time.Duration(sessionMaxAge+sessionKeepAge)*time.Second
}

func (s *session) load() error {
	var r struct {
		Data []byte `gorm:"type:longblob"`
	}

	if err := s.db.Model(&model.SessionRecord{Token: s.t}).Select("data").Scan(&r).Error; err != nil {
		return err
	}

	if err := gob.NewDecoder(bytes.NewReader(r.Data)).Decode(&s.data); err != nil {
		return err
	}

	s.loaded = true
	return nil
}

func (s *session) save() error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(s.data); err != nil {
		panic(err) // gobにdataの中身の構造体が登録されていない
	}
	return s.db.Model(&model.SessionRecord{Token: s.t}).Update("data", buf.Bytes()).Error
}

type cachedSession struct {
	t         string
	refID     uuid.UUID
	userID    uuid.UUID
	createdAt time.Time
}

type sessionStore struct {
	db    *gorm.DB
	cache *lru.Cache
}

func NewGormStore(db *gorm.DB) Store {
	cache, _ := lru.New(cacheSize)
	return &sessionStore{
		db:    db,
		cache: cache,
	}
}

func (ss *sessionStore) GetSession(c echo.Context, createIfNotExist bool) (Session, error) {
	var token string
	cookie, err := c.Cookie(CookieName)
	if err == nil {
		token = cookie.Value
	}

	var s Session
	if len(token) > 0 {
		s, err = ss.GetSessionByToken(token)
		if err != nil && err != ErrSessionNotFound {
			return nil, err
		}
	}

	if s != nil {
		if !s.Expired() {
			return s, nil
		}
		if s.Refreshable() {
			return ss.RenewSession(c, s.UserID())
		}
	}

	if !createIfNotExist {
		return nil, ss.RevokeSession(c)
	}

	// セッション発行
	return ss.RenewSession(c, uuid.Nil)
}

func (ss *sessionStore) GetSessionByToken(token string) (Session, error) {
	if len(token) == 0 {
		return nil, ErrSessionNotFound
	}

	if _v, ok := ss.cache.Get(token); ok {
		v := _v.(*cachedSession)
		return newSession(ss.db, v.t, v.refID, v.userID, v.createdAt, nil), nil
	}

	var r model.SessionRecord
	err := ss.db.First(&r, &model.SessionRecord{Token: token}).Error
	if err == nil {
		if r.UserID != uuid.Nil {
			ss.cache.Add(r.Token, &cachedSession{t: r.Token, refID: r.ReferenceID, userID: r.UserID, createdAt: r.Created})
		}

		data, err := r.GetData()
		if err != nil {
			return nil, err
		}
		return newSession(ss.db, r.Token, r.ReferenceID, r.UserID, r.Created, data), nil
	}

	if gorm.ErrRecordNotFound == err {
		return nil, ErrSessionNotFound
	}
	return nil, err
}

func (ss *sessionStore) GetSessionsByUserID(userID uuid.UUID) ([]Session, error) {
	if userID == uuid.Nil {
		return []Session{}, nil
	}

	var records []*model.SessionRecord
	if err := ss.db.Find(&records, &model.SessionRecord{UserID: userID}).Error; err != nil {
		return nil, err
	}

	result := make([]Session, 0)
	for _, r := range records {
		data, err := r.GetData()
		if err != nil {
			return nil, err
		}
		s := newSession(ss.db, r.Token, r.ReferenceID, r.UserID, r.Created, data)
		if s.Refreshable() {
			result = append(result, s)
		}
	}
	return result, nil
}

func (ss *sessionStore) RevokeSession(c echo.Context) error {
	cookie, err := c.Cookie(CookieName)
	if err != nil {
		return nil
	}

	if err := ss.db.Delete(&model.SessionRecord{Token: cookie.Value}).Error; err != nil {
		return err
	}
	ss.cache.Remove(cookie.Value)

	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	return nil
}

func (ss *sessionStore) RevokeSessionByRefID(refID uuid.UUID) error {
	if refID == uuid.Nil {
		return nil
	}

	var r model.SessionRecord
	if err := ss.db.First(&r, &model.SessionRecord{ReferenceID: refID}).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			return nil
		}
		return err
	}
	if err := ss.db.Delete(&model.SessionRecord{Token: r.Token}).Error; err != nil {
		return err
	}
	ss.cache.Remove(r.Token)

	return nil
}

func (ss *sessionStore) RevokeSessionsByUserID(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return nil
	}

	var rs []*model.SessionRecord
	if err := ss.db.Find(&rs, &model.SessionRecord{UserID: userID}).Error; err != nil {
		return err
	}
	if err := ss.db.Delete(&model.SessionRecord{}, "user_id = ?", userID).Error; err != nil {
		return err
	}

	for _, r := range rs {
		ss.cache.Remove(r.Token)
	}
	return nil
}

func (ss *sessionStore) RenewSession(c echo.Context, userID uuid.UUID) (Session, error) {
	cookie, _ := c.Cookie(CookieName)
	if cookie != nil && len(cookie.Value) > 0 {
		if err := ss.db.Delete(&model.SessionRecord{Token: cookie.Value}).Error; err != nil {
			return nil, err
		}
		ss.cache.Remove(cookie.Value)
	} else {
		cookie = &http.Cookie{}
	}

	s, err := ss.IssueSession(userID, nil)
	if err != nil {
		return nil, err
	}

	cookie.Name = CookieName
	cookie.Value = s.Token()
	cookie.Expires = time.Now().Add(time.Duration(sessionMaxAge+sessionKeepAge) * time.Second)
	cookie.MaxAge = sessionMaxAge + sessionKeepAge
	cookie.Path = "/"
	cookie.HttpOnly = true
	c.SetCookie(cookie)

	return s, nil
}

func (ss *sessionStore) IssueSession(userID uuid.UUID, data map[string]interface{}) (Session, error) {
	if data == nil {
		data = map[string]interface{}{}
	}

	s := &model.SessionRecord{
		Token:       random.SecureAlphaNumeric(50),
		ReferenceID: uuid.Must(uuid.NewV4()),
		UserID:      userID,
		Created:     time.Now(),
	}
	s.SetData(data)

	if err := ss.db.Create(s).Error; err != nil {
		return nil, err
	}
	ss.cache.Add(s.Token, &cachedSession{
		t:         s.Token,
		refID:     s.ReferenceID,
		userID:    s.UserID,
		createdAt: s.Created,
	})

	return newSession(ss.db, s.Token, s.ReferenceID, s.UserID, s.Created, data), nil
}
