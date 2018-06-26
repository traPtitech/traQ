package model

import (
	"github.com/jinzhu/gorm"
	"time"
)

// OAuth2Authorize OAuth2 認可データの構造体
type OAuth2Authorize struct {
	Code                string `gorm:"size:36;primary_key"`
	ClientID            string `gorm:"type:char(36)"`
	UserID              string `gorm:"type:char(36)"`
	ExpiresIn           int
	RedirectURI         string    `gorm:"type:text"`
	Scopes              string    `gorm:"type:text"`
	OriginalScopes      string    `gorm:"type:text"`
	CodeChallenge       string    `gorm:"size:128"`
	CodeChallengeMethod string    `gorm:"type:text"`
	Nonce               string    `gorm:"type:text"`
	CreatedAt           time.Time `gorm:"precision:6"`
}

// TableName OAuth2Authorizeのテーブル名
func (*OAuth2Authorize) TableName() string {
	return "oauth2_authorizes"
}

// Create データベースに挿入します
func (a *OAuth2Authorize) Create() error {
	return db.Create(a).Error
}

// Delete データベースから削除します
func (a *OAuth2Authorize) Delete() error {
	return db.Delete(a).Error
}

// GetOAuth2Authorize 認可コードからOAuth2Authorizeを取得します
func GetOAuth2Authorize(code string) (oa *OAuth2Authorize, err error) {
	oa = &OAuth2Authorize{}
	err = db.Where(OAuth2Authorize{Code: code}).Take(oa).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return
}
