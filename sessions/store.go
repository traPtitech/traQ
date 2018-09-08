package sessions

import (
	"bytes"
	"encoding/gob"
	"fmt"
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

// Store セッション永続化ストア
type Store interface {
	GetByID(id string) (*Session, error)
	DestroyByID(id string) error
	Save(id string, session *Session) error
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
	db *gorm.DB
}

// GetByID gets id's session
func (s *GORMStore) GetByID(id string) (*Session, error) {
	var record SessionRecord
	if err := s.db.First(&record, &SessionRecord{ID: id}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return record.decode()
}

// DestroyByID deletes id's session
func (s *GORMStore) DestroyByID(id string) error {
	return s.db.Delete(&SessionRecord{ID: id}).Error
}

// Save saves id's session
func (s *GORMStore) Save(id string, session *Session) error {
	sr := &SessionRecord{}
	sr.encode(session)
	return s.db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE user_id = VALUES(user_id), last_access = VALUES(last_access), last_ip = VALUES(last_ip), last_user_agent = VALUES(last_user_agent), data = VALUES(data)").
		Create(sr).
		Error
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

	return &GORMStore{db: db}, nil
}
