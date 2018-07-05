package model

import (
	"github.com/jinzhu/gorm"
	"time"
)

// OAuth2Token : OAuth2 トークンの構造体
type OAuth2Token struct {
	ID           string `gorm:"type:char(36);primary_key"`
	ClientID     string `gorm:"type:char(36)"`
	UserID       string `gorm:"type:char(36)"`
	RedirectURI  string `gorm:"type:text"`
	AccessToken  string `gorm:"type:varchar(36);unique"`
	RefreshToken string `gorm:"type:varchar(36);unique"`
	Scopes       string `gorm:"type:text"`
	ExpiresIn    int
	CreatedAt    time.Time  `gorm:"precision:6"`
	DeletedAt    *time.Time `gorm:"precision:6"`
}

// TableName : OAuth2Tokenのテーブル名
func (*OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// Create : データベースに挿入します
func (t *OAuth2Token) Create() error {
	return db.Create(t).Error
}

// Delete : データベースから削除します
func (t *OAuth2Token) Delete() error {
	return db.Delete(t).Error
}

// GetOAuth2TokenByRefresh : リフレッシュトークンからOAuth2Tokenを取得します
func GetOAuth2TokenByRefresh(refresh string) (ot *OAuth2Token, err error) {
	ot = &OAuth2Token{}
	err = db.Where(OAuth2Token{RefreshToken: refresh}).Take(ot).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return
}

// GetOAuth2TokenByUser : 指定したユーザーIDのOAuth2Tokenを全て取得します
func GetOAuth2TokenByUser(userID string) (ts []*OAuth2Token, err error) {
	err = db.Where(OAuth2Token{UserID: userID}).Find(&ts).Error
	return
}

// GetOAuth2TokenByClient : 指定したクライアントのOAuth2Tokenを全て取得します
func GetOAuth2TokenByClient(clientID string) (ts []*OAuth2Token, err error) {
	err = db.Where(OAuth2Token{ClientID: clientID}).Find(&ts).Error
	return
}

// GetOAuth2TokenByAccess : アクセストークンからOAuth2Tokenを取得します
func GetOAuth2TokenByAccess(access string) (ot *OAuth2Token, err error) {
	ot = &OAuth2Token{}
	err = db.Where(OAuth2Token{AccessToken: access}).Take(ot).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return
}

// GetOAUth2TokenByID : トークンIDからOAuth2Tokenを取得します
func GetOAUth2TokenByID(id string) (ot *OAuth2Token, err error) {
	ot = &OAuth2Token{}
	err = db.Where(OAuth2Token{ID: id}).Take(ot).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return
}
