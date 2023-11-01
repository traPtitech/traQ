package model

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/utils/validator"
)

// AccessScope クライアントのスコープ
//
// AccessScopeに使用可能な文字のASCIIコードは次の通りです。
//
// %x21, %x23-5B, %x5D-7E
//
// /と"は使えません。
type AccessScope string

// AccessScopes AccessScopeのセット
type AccessScopes map[AccessScope]struct{}

// SupportedAccessScopes 対応するスコープ一覧を返します
func SupportedAccessScopes() []string {
	return []string{"read", "write", "manage_bot", "openid", "profile", "email"}
}

// Value database/sql/driver.Valuer 実装
func (arr AccessScopes) Value() (driver.Value, error) {
	return arr.String(), nil
}

// Scan database/sql.Scanner 実装
func (arr *AccessScopes) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		*arr = AccessScopes{}
	case string:
		arr.FromString(s)
	case []byte:
		arr.FromString(string(s))
	default:
		return errors.New("failed to scan AccessScopes")
	}
	return nil
}

// MarshalJSON encoding/json.Marshaler 実装
func (arr *AccessScopes) MarshalJSON() ([]byte, error) {
	return json.Marshal(arr.StringArray())
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (arr *AccessScopes) UnmarshalJSON(data []byte) error {
	var str []string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}

	s := AccessScopes{}
	for _, v := range str {
		s.Add(AccessScope(v))
	}
	*arr = s
	return nil
}

// FromString スペース区切り文字列からAccessScopeを抽出して追加します
func (arr *AccessScopes) FromString(s string) {
	r := AccessScopes{}
	for _, v := range strings.Fields(s) {
		r.Add(AccessScope(v))
	}
	*arr = r
}

// Add AccessScopesにスコープを加えます
func (arr AccessScopes) Add(s ...AccessScope) {
	for _, v := range s {
		arr[v] = struct{}{}
	}
}

// Contains AccessScopesに指定したスコープが含まれるかどうかを返します
func (arr AccessScopes) Contains(s AccessScope) bool {
	_, ok := arr[s]
	return ok
}

// String AccessScopesをスペース区切りで文字列に出力します
func (arr AccessScopes) String() string {
	sa := make([]string, 0, len(arr))
	for s := range arr {
		sa = append(sa, string(s))
	}
	return strings.Join(sa, " ")
}

// StringArray AccessScopesをstringの配列に変換します
func (arr AccessScopes) StringArray() (r []string) {
	r = make([]string, 0, len(arr))
	for s := range arr {
		r = append(r, string(s))
	}
	return r
}

// Validate github.com/go-ozzo/ozzo-validation.Validatable 実装
func (arr AccessScopes) Validate() error {
	// TODO カスタムスコープに対応
	scopes := lo.Map(SupportedAccessScopes(), func(s string, _ int) any { return s })
	return vd.Validate(arr.StringArray(), vd.Each(vd.Required, vd.In(scopes...)))
}

// OAuth2Authorize OAuth2 認可データの構造体
type OAuth2Authorize struct {
	Code                string    `gorm:"type:varchar(36);primaryKey"`
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
	ID           string `gorm:"type:char(36);primaryKey"`
	Name         string `gorm:"type:varchar(32)"`
	Description  string `gorm:"type:text"`
	Confidential bool
	CreatorID    uuid.UUID      `gorm:"type:char(36)"`
	Secret       string         `gorm:"type:varchar(36)"`
	RedirectURI  string         `gorm:"type:text"`
	Scopes       AccessScopes   `gorm:"type:text"`
	CreatedAt    time.Time      `gorm:"precision:6"`
	UpdatedAt    time.Time      `gorm:"precision:6"`
	DeletedAt    gorm.DeletedAt `gorm:"precision:6"`
}

// TableName OAuth2Clientのテーブル名
func (*OAuth2Client) TableName() string {
	return "oauth2_clients"
}

// GetAvailableScopes requestで与えられたスコープのうち、利用可能なものを返します
func (c *OAuth2Client) GetAvailableScopes(request AccessScopes) (result AccessScopes) {
	result = AccessScopes{}
	for s := range request {
		if c.Scopes.Contains(s) {
			result.Add(s)
		}
	}
	return
}

// OAuth2Token OAuth2 トークンの構造体
type OAuth2Token struct {
	ID             uuid.UUID    `gorm:"type:char(36);primaryKey"`
	ClientID       string       `gorm:"type:char(36)"`
	UserID         uuid.UUID    `gorm:"type:char(36)"`
	RedirectURI    string       `gorm:"type:text"`
	AccessToken    string       `gorm:"type:varchar(36);unique"`
	RefreshToken   string       `gorm:"type:varchar(36);unique"`
	RefreshEnabled bool         `gorm:"type:boolean;default:false"`
	Scopes         AccessScopes `gorm:"type:text"`
	ExpiresIn      int
	CreatedAt      time.Time      `gorm:"precision:6"`
	DeletedAt      gorm.DeletedAt `gorm:"precision:6"`
}

// TableName OAuth2Tokenのテーブル名
func (*OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// GetAvailableScopes requestで与えられたスコープのうち、利用可能なものを返します
func (t *OAuth2Token) GetAvailableScopes(request AccessScopes) (result AccessScopes) {
	result = AccessScopes{}
	for s := range request {
		if t.Scopes.Contains(s) {
			result.Add(s)
		}
	}
	return
}

func (t *OAuth2Token) Deadline() time.Time {
	return t.CreatedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// IsExpired 有効期限が切れているかどうか
func (t *OAuth2Token) IsExpired() bool {
	return t.Deadline().Before(time.Now())
}

// IsRefreshEnabled リフレッシュトークンが有効かどうか
func (t *OAuth2Token) IsRefreshEnabled() bool {
	return t.RefreshEnabled && len(t.RefreshToken) != 0
}
