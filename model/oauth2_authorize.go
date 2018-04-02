package model

import (
	"time"
)

// OAuth2Authorize : OAuth2 認可データの構造体
type OAuth2Authorize struct {
	Code                string    `xorm:"varchar(36) not null pk"`
	ClientID            string    `xorm:"char(36) not null"`
	UserID              string    `xorm:"char(36) not null"`
	ExpiresIn           int       `xorm:"int not null"`
	RedirectURI         string    `xorm:"text not null"`
	Scopes              string    `xorm:"text not null"`
	OriginalScopes      string    `xorm:"text not null"`
	CodeChallenge       string    `xorm:"varchar(128) not null"`
	CodeChallengeMethod string    `xorm:"text not null"`
	Nonce               string    `xorm:"text not null"`
	CreatedAt           time.Time `xorm:"timestamp not null"`
}

// TableName : OAuth2Authorizeのテーブル名
func (*OAuth2Authorize) TableName() string {
	return "oauth2_authorizes"
}

// Create : データベースに挿入します
func (a *OAuth2Authorize) Create() error {
	_, err := db.UseBool().MustCols().InsertOne(a)
	return err
}

// Delete : データベースから削除します
func (a *OAuth2Authorize) Delete() error {
	_, err := db.Delete(a)
	return err
}

// GetOAuth2Authorize : 認可コードからOAuth2Authorizeを取得します
func GetOAuth2Authorize(code string) (oa *OAuth2Authorize, err error) {
	oa = &OAuth2Authorize{}
	ok, err := db.Where("code = ?", code).Get(oa)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return
}
