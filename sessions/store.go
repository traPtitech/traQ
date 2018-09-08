package sessions

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"log"
	"sync"
	"time"
)

var store Store

func init() {
	store = NewInMemoryStore()
}

// Store セッションストア
type Store interface {
	GetByToken(token string) (*Session, error)
	GetByUserID(id uuid.UUID) ([]*Session, error)
	GetByReferenceID(id uuid.UUID) (*Session, error)
	DestroyByToken(token string) error
	Save(token string, session *Session) error
}

// CacheableStore キャッシュ可能なセッションストア
type CacheableStore interface {
	Store
	PurgeCache()
}

// SetStore ストアをセットします
func SetStore(s Store) {
	store = s
}

// InMemoryStore for test use
type InMemoryStore struct {
	sync.RWMutex
	sessions map[string]*Session
}

// NewInMemoryStore インメモリストアを作成します
func NewInMemoryStore() Store {
	return &InMemoryStore{
		sessions: make(map[string]*Session),
	}
}

// GetByToken gets token's session
func (s *InMemoryStore) GetByToken(token string) (*Session, error) {
	s.RLock()
	defer s.RUnlock()
	sess, _ := s.sessions[token]
	return sess, nil
}

// GetByUserID gets the user's sessions
func (s *InMemoryStore) GetByUserID(id uuid.UUID) (result []*Session, err error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.sessions {
		if uuid.Equal(v.userID, id) {
			result = append(result, v)
		}
	}
	return
}

// GetByReferenceID gets id's session
func (s *InMemoryStore) GetByReferenceID(id uuid.UUID) (*Session, error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.sessions {
		if uuid.Equal(v.referenceID, id) {
			return v, nil
		}
	}
	return nil, nil
}

// DestroyByToken deletes token's session
func (s *InMemoryStore) DestroyByToken(token string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, token)
	return nil
}

// Save saves token's session
func (s *InMemoryStore) Save(token string, session *Session) error {
	s.Lock()
	defer s.Unlock()
	s.sessions[token] = session
	return nil
}

// GORMStore GORMストア
type GORMStore struct {
	sync.Mutex
	db    *gorm.DB
	cache *lru.Cache
}

// GetByToken gets token's session
func (s *GORMStore) GetByToken(token string) (*Session, error) {
	s.Lock()
	defer s.Unlock()

	if session, ok := s.cache.Get(token); ok {
		return session.(*Session), nil
	}

	var record SessionRecord
	if err := s.db.First(&record, &SessionRecord{Token: token}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	session, err := record.decode()
	if err != nil {
		return nil, err
	}

	s.cache.Add(token, session)
	return session, nil
}

// GetByUserID gets the user's sessions
func (s *GORMStore) GetByUserID(id uuid.UUID) ([]*Session, error) {
	var records []*SessionRecord
	if err := s.db.Where(&SessionRecord{UserID: id.String()}).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]*Session, len(records))

	s.Lock()
	defer s.Unlock()
	for k, v := range records {
		if sess, ok := s.cache.Get(v.Token); ok {
			result[k] = sess.(*Session)
		} else {
			sess, err := v.decode()
			if err != nil {
				result[k] = sess
			} else {
				return nil, err
			}
		}
	}
	return result, nil
}

// GetByReferenceID gets id's session
func (s *GORMStore) GetByReferenceID(id uuid.UUID) (*Session, error) {
	var record SessionRecord
	if err := s.db.First(&record, &SessionRecord{ReferenceID: id.String()}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	s.Lock()
	defer s.Unlock()
	if sess, ok := s.cache.Get(record.Token); ok {
		return sess.(*Session), nil
	}
	return record.decode()
}

// DestroyByToken deletes token's session
func (s *GORMStore) DestroyByToken(token string) error {
	s.Lock()
	defer s.Unlock()
	s.cache.Remove(token)
	return s.db.Delete(&SessionRecord{Token: token}).Error
}

// Save saves token's session
func (s *GORMStore) Save(token string, session *Session) error {
	session.Lock()
	session.lastAccess = time.Now()
	session.Unlock()

	s.Lock()
	defer s.Unlock()
	s.cache.Add(token, session)

	sr := &SessionRecord{}
	sr.encode(session)
	return s.db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE user_id = VALUES(user_id), last_access = VALUES(last_access), last_ip = VALUES(last_ip), last_user_agent = VALUES(last_user_agent), data = VALUES(data)").
		Create(sr).
		Error
}

// PurgeCache キャッシュを全て永続化してから解放します
func (s *GORMStore) PurgeCache() {
	s.Lock()
	defer s.Unlock()
	s.cache.Purge()
}

// SessionRecord GORM用Session構造体
type SessionRecord struct {
	Token         string    `gorm:"type:varchar(50);primary_key"`
	ReferenceID   string    `gorm:"type:char(36);unique"`
	UserID        string    `gorm:"type:varchar(36);index"`
	LastAccess    time.Time `gorm:"precision:6"`
	LastIP        string    `gorm:"type:text"`
	LastUserAgent string    `gorm:"type:text"`
	Data          []byte    `gorm:"type:longblob"`
	Created       time.Time `gorm:"precision:6"`
}

// TableName SessionRecordのテーブル名
func (*SessionRecord) TableName() string {
	return tableName
}

func (sr *SessionRecord) encode(session *Session) {
	session.RLock()
	defer session.RUnlock()

	sr.Token = session.token
	sr.ReferenceID = session.referenceID.String()
	sr.UserID = session.userID.String()
	sr.LastAccess = session.lastAccess
	sr.LastIP = session.lastIP
	sr.LastUserAgent = session.lastUserAgent
	sr.Created = session.created

	buffer := bytes.Buffer{}
	if err := gob.NewEncoder(&buffer).Encode(session.data); err != nil {
		log.Fatal(err) // gobにdataの中身の構造体が登録されていない
	}
	sr.Data = buffer.Bytes()
}

func (sr *SessionRecord) decode() (*Session, error) {
	s := &Session{
		token:         sr.Token,
		referenceID:   uuid.Must(uuid.FromString(sr.ReferenceID)),
		userID:        uuid.FromStringOrNil(sr.UserID),
		lastAccess:    sr.LastAccess,
		lastIP:        sr.LastIP,
		lastUserAgent: sr.LastUserAgent,
		created:       sr.Created,
	}

	if err := gob.NewDecoder(bytes.NewReader(sr.Data)).Decode(&s.data); err != nil {
		return nil, err
	}

	return s, nil
}

// NewGORMStore GORMストアを作成します
func NewGORMStore(db *gorm.DB) (Store, error) {
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(&SessionRecord{}).Error; err != nil {
		return nil, fmt.Errorf("failed to sync Table schema: %v", err)
	}

	cache, err := lru.NewWithEvict(cacheSize, func(key interface{}, value interface{}) {
		sess := value.(*Session)
		if !sess.Expired() {
			sr := &SessionRecord{}
			sr.encode(sess)
			db.Save(sr)
		}
	})
	if err != nil {
		return nil, err
	}

	return &GORMStore{db: db, cache: cache}, nil
}
