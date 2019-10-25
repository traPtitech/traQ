package sessions

import (
	"encoding/gob"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/utils"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var mutexes *utils.KeyMutex

// Session セッション構造体
type Session struct {
	token         string
	referenceID   uuid.UUID
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
	mutexes = utils.NewKeyMutex(mutexSize)
}

// Get セッションを取得します
func Get(rw http.ResponseWriter, req *http.Request, createIfNotExists bool) (*Session, error) {
	userAgent := req.Header.Get("User-Agent")
	ip := realIP(req)

	var token string
	cookie, err := req.Cookie(CookieName)
	if err == nil {
		token = cookie.Value
	}

	var session *Session
	if len(token) > 0 {
		mutexes.Lock(token)
		defer mutexes.Unlock(token)

		var err error
		session, err = store.GetByToken(token)
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
		absent := time.Since(session.lastAccess)
		session.RUnlock()

		// TODO ipアドレス確認をする (地域・国レベルでのipアドレス変化を検出)
		valid := age <= time.Duration(sessionMaxAge)*time.Second
		regenerate := absent <= time.Duration(sessionKeepAge)*time.Second

		if !valid {

			if regenerate {
				// 最終アクセスからsessionKeepAge経過していない場合はセッションを継続

				uid := session.GetUserID()
				err := session.Destroy(rw, req)
				if err != nil {
					return nil, err
				}
				session, err := IssueNewSession(ip, userAgent)
				if err != nil {
					return nil, err
				}
				err = session.SetUser(uid)
				if err != nil {
					return nil, err
				}
				setCookie(session.token, rw)
				return session, nil
			}

			if err := session.Destroy(rw, req); err != nil {
				return nil, err
			}

			session = nil

		} else {
			session.Lock()
			session.lastAccess = time.Now()
			session.lastUserAgent = userAgent
			session.lastIP = ip
			session.Unlock()

			return session, nil
		}
	}

	if !createIfNotExists {
		return nil, nil
	}

	session, err = IssueNewSession(ip, userAgent)
	if err != nil {
		return nil, nil
	}
	setCookie(session.token, rw)

	return session, nil
}

// IssueNewSession 新しいセッションを生成します
func IssueNewSession(ip string, userAgent string) (s *Session, err error) {
	session := &Session{
		token:         utils.RandAlphabetAndNumberString(50),
		referenceID:   uuid.Must(uuid.NewV4()),
		userID:        uuid.Nil,
		created:       time.Now(),
		lastAccess:    time.Now(),
		lastUserAgent: userAgent,
		lastIP:        ip,
		data:          make(map[string]interface{}),
	}
	if err := store.Save(session.token, session); err != nil {
		return nil, err
	}
	return session, nil
}

// GetByToken 指定したtokenのセッションを取得します
func GetByToken(token string) (s *Session, err error) {
	mutexes.Lock(token)
	defer mutexes.Unlock(token)

	s, err = store.GetByToken(token)
	if err != nil {
		return nil, err
	}

	if s != nil {
		if s.Expired() {
			if err := DestroyByToken(token); err != nil {
				return nil, err
			}
			s = nil
		}
	}

	return s, nil
}

// GetByUserID 指定したユーザーのセッションを全て取得します
func GetByUserID(id uuid.UUID) ([]*Session, error) {
	sessions, err := store.GetByUserID(id)
	if err != nil {
		return nil, err
	}

	var result []*Session
	for _, v := range sessions {
		mutexes.Lock(v.token)
		if v.Expired() {
			_ = DestroyByToken(v.token)
		} else {
			result = append(result, v)
		}
		mutexes.Unlock(v.token)
	}

	return result, nil
}

// DestroyByToken 指定したtokenのセッションを破棄します
func DestroyByToken(token string) error {
	return store.DestroyByToken(token)
}

// DestroyByUserID 指定したユーザーのセッションを全て破棄します
func DestroyByUserID(id uuid.UUID) error {
	sessions, err := store.GetByUserID(id)
	if err != nil {
		return err
	}

	for _, v := range sessions {
		mutexes.Lock(v.token)
		if err := DestroyByToken(v.token); err != nil {
			mutexes.Unlock(v.token)
			return err
		}
		mutexes.Unlock(v.token)
	}

	return nil
}

// DestroyByReferenceID 指定したユーザーのreferenceIDのセッションを破棄します
func DestroyByReferenceID(userID, referenceID uuid.UUID) error {
	session, err := store.GetByReferenceID(referenceID)
	if err != nil {
		return err
	}
	if session.userID != userID {
		return nil
	}
	mutexes.Lock(session.token)
	defer mutexes.Unlock(session.token)

	return DestroyByToken(session.token)
}

// Destroy セッションを破棄します
func (s *Session) Destroy(rw http.ResponseWriter, req *http.Request) error {
	if err := DestroyByToken(s.token); err != nil {
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
func (s *Session) GetSessionInfo() (referenceID uuid.UUID, created, lastAccess time.Time, lastIP, lastUserAgent string) {
	s.RLock()
	defer s.RUnlock()
	return s.referenceID, s.created, s.lastAccess, s.lastIP, s.lastUserAgent
}

// SetUser セッションにユーザーを紐づけます
func (s *Session) SetUser(userID uuid.UUID) error {
	s.Lock()
	s.userID = userID
	s.Unlock()
	return store.Save(s.token, s)
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
	return store.Save(s.token, s)
}

// Delete セッションから値を削除します
func (s *Session) Delete(key string) error {
	s.Lock()
	delete(s.data, key)
	s.Unlock()
	return store.Save(s.token, s)
}

// Expired セッションの有効期限が切れているかどうか
func (s *Session) Expired() bool {
	s.RLock()
	age := time.Since(s.created)
	s.RUnlock()
	return age > time.Duration(sessionMaxAge+sessionKeepAge)*time.Second
}

func deleteCookie(cookie *http.Cookie, rw http.ResponseWriter) {
	deleted := *cookie
	deleted.Value = ""
	deleted.Expires = time.Unix(0, 0)
	deleted.MaxAge = -1
	http.SetCookie(rw, &deleted)
}

func setCookie(token string, rw http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Expires:  time.Now().Add(time.Duration(sessionMaxAge+sessionKeepAge) * time.Second),
		MaxAge:   sessionMaxAge + sessionKeepAge,
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(rw, cookie)
}

// PurgeCache キャッシュを全て解放し、その内容を永続化します
func PurgeCache() {
	if s, ok := store.(CacheableStore); ok {
		s.PurgeCache()
	}
}

func realIP(req *http.Request) string {
	if ip := req.Header.Get(echo.HeaderXForwardedFor); ip != "" {
		return strings.Split(ip, ", ")[0]
	}
	if ip := req.Header.Get(echo.HeaderXRealIP); ip != "" {
		return ip
	}
	ra, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ra
}
