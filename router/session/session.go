package session

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
)

const (
	// CookieName セッションクッキー名
	CookieName     = "r_session"
	sessionMaxAge  = 60 * 60 * 24 * 14 // 2 weeks
	sessionKeepAge = 60 * 60 * 24 * 14 // 2 weeks
	cacheSize      = 2048
)

var ErrSessionNotFound = errors.New("session not found")

type Session interface {
	Token() string
	RefID() uuid.UUID
	UserID() uuid.UUID
	CreatedAt() time.Time
	LoggedIn() bool

	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Delete(key string) error

	Expired() bool
	Refreshable() bool
}

type Store interface {
	GetSession(c echo.Context) (Session, error)
	GetSessionByToken(token string) (Session, error)
	GetSessionsByUserID(userID uuid.UUID) ([]Session, error)
	RevokeSession(c echo.Context) error
	RevokeSessionByRefID(refID uuid.UUID) error
	RevokeSessionsByUserID(userID uuid.UUID) error
	RenewSession(c echo.Context, userID uuid.UUID) (Session, error)
	IssueSession(userID uuid.UUID, data map[string]interface{}) (Session, error)
}
