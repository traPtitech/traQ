package model

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"strings"
	"time"
)

// AccessScope クライアントのスコープ
//
// AccessScopeに使用可能な文字のASCIIコードは次の通りです。
//
// %x21, %x23-5B, %x5D-7E
//
// /と"は使えません。
type AccessScope string

// AccessScopes AccessScopeのスライス
type AccessScopes []AccessScope

// Value database/sql/driver.Valuer 実装
func (arr AccessScopes) Value() (driver.Value, error) {
	return arr.String(), nil
}

// Scan database/sql.Scanner 実装
func (arr *AccessScopes) Scan(src interface{}) error {
	if src == nil {
		*arr = AccessScopes{}
		return nil
	}
	if sv, err := driver.String.ConvertValue(src); err == nil {
		if v, ok := sv.(string); ok {
			as := AccessScopes{}
			for _, v := range strings.Split(v, " ") {
				as = append(as, AccessScope(v))
			}
			*arr = as
			return nil
		}
	}
	return errors.New("failed to scan AccessScopes")
}

// Contains AccessScopesに指定したスコープが含まれるかどうかを返します
func (arr AccessScopes) Contains(s AccessScope) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// String AccessScopesをスペース区切りで文字列に出力します
func (arr AccessScopes) String() string {
	sa := make([]string, len(arr))
	for i, v := range arr {
		sa[i] = string(v)
	}
	return strings.Join(sa, " ")
}

// OAuth2Authorize OAuth2 認可データの構造体
type OAuth2Authorize struct {
	Code                string    `gorm:"type:varchar(36);primary_key"`
	ClientID            string    `gorm:"type:char(36)"`
	UserID              uuid.UUID `gorm:"type:char(36)"`
	ExpiresIn           int
	RedirectURI         string       `gorm:"type:text"`
	Scopes              AccessScopes `gorm:"type:text"`
	OriginalScopes      AccessScopes `gorm:"type:text"`
	CodeChallenge       string       `gorm:"type:varchar(128)"`
	CodeChallengeMethod string       `gorm:"type:text"`
	Nonce               string       `gorm:"type:text"`
	CreatedAt           time.Time    `gorm:"precision:6"`
}

// TableName OAuth2Authorizeのテーブル名
func (*OAuth2Authorize) TableName() string {
	return "oauth2_authorizes"
}

// IsExpired 有効期限が切れているかどうか
func (data *OAuth2Authorize) IsExpired() bool {
	return data.CreatedAt.Add(time.Duration(data.ExpiresIn) * time.Second).Before(time.Now())
}

// ValidatePKCE PKCEの検証を行う
func (data *OAuth2Authorize) ValidatePKCE(verifier string) (bool, error) {
	if len(verifier) == 0 {
		return len(data.CodeChallenge) == 0, nil
	}
	if !validator.PKCERegex.MatchString(verifier) {
		return false, nil
	}

	if len(data.CodeChallengeMethod) == 0 {
		data.CodeChallengeMethod = "plain"
	}

	switch data.CodeChallengeMethod {
	case "plain":
		return verifier == data.CodeChallenge, nil
	case "S256":
		hash := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(hash[:]) == data.CodeChallenge, nil
	}

	return false, fmt.Errorf("unknown method: %v", data.CodeChallengeMethod)
}

// OAuth2Client OAuth2 クライアント構造体
type OAuth2Client struct {
	ID           string `gorm:"type:char(36);primary_key"`
	Name         string `gorm:"type:varchar(32)"`
	Description  string `gorm:"type:text"`
	Confidential bool
	CreatorID    uuid.UUID    `gorm:"type:char(36)"`
	Secret       string       `gorm:"type:varchar(36)"`
	RedirectURI  string       `gorm:"type:text"`
	Scopes       AccessScopes `gorm:"type:text"`
	CreatedAt    time.Time    `gorm:"precision:6"`
	UpdatedAt    time.Time    `gorm:"precision:6"`
	DeletedAt    *time.Time   `gorm:"precision:6"`
}

// TableName OAuth2Clientのテーブル名
func (*OAuth2Client) TableName() string {
	return "oauth2_clients"
}

// GetAvailableScopes requestで与えられたスコープのうち、利用可能なものを返します
func (c *OAuth2Client) GetAvailableScopes(request AccessScopes) (result AccessScopes) {
	for _, s := range request {
		if c.Scopes.Contains(s) {
			result = append(result, s)
		}
	}
	return
}

// OAuth2Token OAuth2 トークンの構造体
type OAuth2Token struct {
	ID           uuid.UUID    `gorm:"type:char(36);primary_key"`
	ClientID     string       `gorm:"type:char(36)"`
	UserID       uuid.UUID    `gorm:"type:char(36)"`
	RedirectURI  string       `gorm:"type:text"`
	AccessToken  string       `gorm:"type:varchar(36);unique"`
	RefreshToken string       `gorm:"type:varchar(36);unique"`
	Scopes       AccessScopes `gorm:"type:text"`
	ExpiresIn    int
	CreatedAt    time.Time  `gorm:"precision:6"`
	DeletedAt    *time.Time `gorm:"precision:6"`
}

// TableName OAuth2Tokenのテーブル名
func (*OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// GetAvailableScopes requestで与えられたスコープのうち、利用可能なものを返します
func (t *OAuth2Token) GetAvailableScopes(request AccessScopes) (result AccessScopes) {
	for _, s := range request {
		if t.Scopes.Contains(s) {
			result = append(result, s)
		}
	}
	return
}

// IsExpired 有効期限が切れているかどうか
func (t *OAuth2Token) IsExpired() bool {
	return t.CreatedAt.Add(time.Duration(t.ExpiresIn) * time.Second).Before(time.Now())
}
