package sessions

import (
	"encoding/base64"
	"encoding/gob"
	"github.com/satori/go.uuid"
	"github.com/tomasen/realip"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"sync"
	"time"
)

var mutexes *utils.KeyLocker

// Session セッション構造体
type Session struct {
	id            string
	userID        uuid.UUID
	created       time.Time
	lastAccess    time.Time
	lastIP        string
	lastUserAgent string
	data          map[string]interface{}
	sync.RWMutex
}

func init() {
	gob.Register(map[string]interface{}{})
	mutexes = utils.NewKeyLocker()
}

// Get セッションを取得します
func Get(rw http.ResponseWriter, req *http.Request, createIfNotExists bool) (*Session, error) {
	userAgent := req.Header.Get("User-Agent")
	ip := realip.FromRequest(req)

	var id string
	cookie, err := req.Cookie(CookieName)
	if err == nil {
		id = cookie.Value
	}

	var session *Session
	if len(id) > 0 {
		mutexes.Lock(id)
		defer mutexes.Unlock(id)

		var err error
		session, err = sessions.get(id)
		if err != nil {
			return nil, err
		}
		if session == nil {
			deleteCookie(cookie, rw)
		}
	}

	if session != nil {
		session.RLock()
		age := time.Since(session.created)
		session.RUnlock()

		// TODO ipアドレス確認をする (地域・国レベルでのipアドレス変化を検出)
		// TODO セッションリフレッシュ
		valid := age <= time.Duration(sessionMaxAge)*time.Second

		if !valid {
			if err := session.Destroy(rw, req); err != nil {
				return nil, err
			}
			session = nil
		} else {
			session.Lock()
			defer session.Unlock()
			session.lastAccess = time.Now()
			session.lastUserAgent = userAgent
			session.lastIP = ip

			return session, nil
		}
	}

	if !createIfNotExists {
		return nil, nil
	}

	session = &Session{
		id:            generateRandomString() + generateRandomString(),
		userID:        uuid.Nil,
		created:       time.Now(),
		lastAccess:    time.Now(),
		lastUserAgent: userAgent,
		lastIP:        ip,
		data:          make(map[string]interface{}),
	}
	if err := sessions.set(session); err != nil {
		return nil, err
	}
	setCookie(session.id, rw)

	return session, nil
}

// GetByID 指定したidのセッションを取得します
func GetByID(id string) (s *Session, err error) {
	mutexes.Lock(id)
	defer mutexes.Unlock(id)
	return sessions.get(id)
}

// Destroy セッションを破棄します
func (s *Session) Destroy(rw http.ResponseWriter, req *http.Request) error {
	if err := sessions.delete(s.id); err != nil {
		return err
	}

	cookie, err := req.Cookie(CookieName)
	if err != nil {
		return err
	}
	deleteCookie(cookie, rw)

	return nil
}

// GetUserID セッションに紐づけられているユーザーのIDを返します
func (s *Session) GetUserID() uuid.UUID {
	s.RLock()
	defer s.RUnlock()
	return s.userID
}

// GetSessionInfo セッションの情報を返します
func (s *Session) GetSessionInfo() (created, lastAccess time.Time, lastIP, lastUserAgent string) {
	s.RLock()
	defer s.RUnlock()
	return s.created, s.lastAccess, s.lastIP, s.lastUserAgent
}

// SetUser セッションにユーザーを紐づけます
func (s *Session) SetUser(userID uuid.UUID) error {
	s.Lock()
	s.userID = userID
	s.Unlock()
	return store.Save(s.id, s)
}

// Get セッションから値を取り出します
func (s *Session) Get(key string) interface{} {
	s.RLock()
	defer s.RUnlock()
	value, ok := s.data[key]
	if ok {
		return value
	}
	return nil
}

// Set セッションに値をセットします
func (s *Session) Set(key string, value interface{}) error {
	s.Lock()
	s.data[key] = value
	s.Unlock()
	return store.Save(s.id, s)
}

// Delete セッションから値を削除します
func (s *Session) Delete(key string) error {
	s.Lock()
	delete(s.data, key)
	s.Unlock()
	return store.Save(s.id, s)
}

func deleteCookie(cookie *http.Cookie, rw http.ResponseWriter) {
	deleted := *cookie
	deleted.Value = ""
	deleted.Expires = time.Unix(0, 0)
	deleted.MaxAge = -1
	http.SetCookie(rw, &deleted)
}

func setCookie(id string, rw http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Expires:  time.Now().Add(time.Duration(sessionMaxAge) * time.Second),
		MaxAge:   sessionMaxAge,
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(rw, cookie)
}

func generateRandomString() string {
	return base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes())
}

// PurgeCache キャッシュを全て解放し、その内容を永続化します
func PurgeCache() {
	sessions.purge()
}
