package session

import (
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/utils/random"
)

type memorySession struct {
	t         string
	refID     uuid.UUID
	userID    uuid.UUID
	createdAt time.Time
	data      map[string]interface{}
	sync.Mutex
}

func newMemorySession(t string, refID uuid.UUID, userID uuid.UUID, createdAt time.Time, data map[string]interface{}) *memorySession {
	return &memorySession{
		t:         t,
		refID:     refID,
		userID:    userID,
		createdAt: createdAt,
		data:      data,
	}
}

func (s *memorySession) Token() string {
	return s.t
}

func (s *memorySession) RefID() uuid.UUID {
	return s.refID
}

func (s *memorySession) UserID() uuid.UUID {
	return s.userID
}

func (s *memorySession) CreatedAt() time.Time {
	return s.createdAt
}

func (s *memorySession) LoggedIn() bool {
	return s.userID != uuid.Nil
}

func (s *memorySession) Get(key string) (interface{}, error) {
	s.Lock()
	defer s.Unlock()
	return s.data[key], nil
}

func (s *memorySession) Set(key string, value interface{}) error {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
	return nil
}

func (s *memorySession) Delete(key string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.data, key)
	return nil
}

func (s *memorySession) Expired() bool {
	return time.Since(s.createdAt) > time.Duration(sessionMaxAge)*time.Second
}

func (s *memorySession) Refreshable() bool {
	return time.Since(s.createdAt) <= time.Duration(sessionMaxAge+sessionKeepAge)*time.Second
}

type memoryStore struct {
	sessions map[string]*memorySession
	sync.RWMutex
}

func NewMemorySessionStore() Store {
	return &memoryStore{
		sessions: map[string]*memorySession{},
	}
}

func (ms *memoryStore) GetSession(c echo.Context) (Session, error) {
	var token string
	cookie, err := c.Cookie(CookieName)
	if err == nil {
		token = cookie.Value
	}

	var s Session
	if len(token) > 0 {
		s, err = ms.GetSessionByToken(token)
		if err != nil && err != ErrSessionNotFound {
			return nil, err
		}
	}

	if s != nil {
		if !s.Expired() {
			return s, nil
		}
		if s.Refreshable() {
			return ms.RenewSession(c, s.UserID())
		}
	}

	return nil, ms.RevokeSession(c)
}

func (ms *memoryStore) GetSessionByToken(token string) (Session, error) {
	if len(token) == 0 {
		return nil, ErrSessionNotFound
	}

	ms.RLock()
	defer ms.RUnlock()
	s, ok := ms.sessions[token]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

func (ms *memoryStore) GetSessionsByUserID(userID uuid.UUID) ([]Session, error) {
	if userID == uuid.Nil {
		return []Session{}, nil
	}

	ms.RLock()
	defer ms.RUnlock()
	result := make([]Session, 0)
	for _, s := range ms.sessions {
		if s.UserID() == userID && s.Refreshable() {
			result = append(result, s)
		}
	}
	return result, nil
}

func (ms *memoryStore) RevokeSession(c echo.Context) error {
	cookie, err := c.Cookie(CookieName)
	if err != nil {
		return nil
	}
	if len(cookie.Value) == 0 {
		return nil
	}

	ms.Lock()
	delete(ms.sessions, cookie.Value)
	ms.Unlock()

	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	return nil
}

func (ms *memoryStore) RevokeSessionByRefID(refID uuid.UUID) error {
	if refID == uuid.Nil {
		return nil
	}
	ms.Lock()
	defer ms.Unlock()
	for k, s := range ms.sessions {
		if s.RefID() == refID {
			delete(ms.sessions, k)
			return nil
		}
	}
	return nil
}

func (ms *memoryStore) RevokeSessionsByUserID(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return nil
	}
	ms.Lock()
	defer ms.Unlock()
	for k, s := range ms.sessions {
		if s.UserID() == userID {
			delete(ms.sessions, k)
		}
	}
	return nil
}

func (ms *memoryStore) RenewSession(c echo.Context, userID uuid.UUID) (Session, error) {
	cookie, _ := c.Cookie(CookieName)
	if cookie != nil && len(cookie.Value) > 0 {
		ms.Lock()
		delete(ms.sessions, cookie.Value)
		ms.Unlock()
	} else {
		cookie = &http.Cookie{}
	}

	s, err := ms.IssueSession(userID, nil)
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

func (ms *memoryStore) IssueSession(userID uuid.UUID, data map[string]interface{}) (Session, error) {
	if data == nil {
		data = map[string]interface{}{}
	}
	s := newMemorySession(random.SecureAlphaNumeric(50), uuid.Must(uuid.NewV7()), userID, time.Now(), data)
	ms.Lock()
	ms.sessions[s.Token()] = s
	ms.Unlock()
	return s, nil
}
