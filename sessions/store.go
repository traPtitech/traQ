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
	GetByID(id string) (*Session, error)
	GetByUserID(id uuid.UUID) ([]*Session, error)
	DestroyByID(id string) error
	Save(id string, session *Session) error
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

// GetByID gets id's session
func (s *InMemoryStore) GetByID(id string) (*Session, error) {
	s.RLock()
	defer s.RUnlock()
	sess, _ := s.sessions[id]
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

// DestroyByID deletes id's session
func (s *InMemoryStore) DestroyByID(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, id)
	return nil
}

// Save saves id's session
func (s *InMemoryStore) Save(id string, session *Session) error {
	s.Lock()
	defer s.Unlock()
	s.sessions[id] = session
	return nil
}

// GORMStore GORMストア
type GORMStore struct {
	sync.Mutex
	db    *gorm.DB
	cache *lru.Cache
}

// GetByID gets id's session
func (s *GORMStore) GetByID(id string) (*Session, error) {
	s.Lock()
	defer s.Unlock()

	if session, ok := s.cache.Get(id); ok {
		return session.(*Session), nil
	}

	var record SessionRecord
	if err := s.db.First(&record, &SessionRecord{ID: id}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	session, err := record.decode()
	if err != nil {
		return nil, err
	}

	s.cache.Add(id, session)
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
		if sess, ok := s.cache.Get(v.ID); ok {
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

// DestroyByID deletes id's session
func (s *GORMStore) DestroyByID(id string) error {
	s.Lock()
	defer s.Unlock()
	s.cache.Remove(id)
	return s.db.Delete(&SessionRecord{ID: id}).Error
}

// Save saves id's session
func (s *GORMStore) Save(id string, session *Session) error {
	session.Lock()
	session.lastAccess = time.Now()
	session.Unlock()

	s.Lock()
	defer s.Unlock()
	s.cache.Add(id, session)

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
	ID            string    `gorm:"type:varchar(50);primary_key"`
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

	sr.ID = session.id
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
		id:            sr.ID,
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
