package model

import (
	"time"
)

// OAuth2Token : OAuth2 トークンの構造体
type OAuth2Token struct {
	ID           string    `xorm:"char(36) not null pk"`
	ClientID     string    `xorm:"char(36) not null"`
	UserID       string    `xorm:"char(36) not null"`
	RedirectURI  string    `xorm:"text not null"`
	AccessToken  string    `xorm:"varchar(36) not null"`
	RefreshToken string    `xorm:"varchar(36) not null"`
	Scopes       string    `xorm:"text not null"`
	ExpiresIn    int       `xorm:"int not null"`
	CreatedAt    time.Time `xorm:"timestamp not null"`
}

// TableName : OAuth2Tokenのテーブル名
func (*OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// Create : データベースに挿入します
func (t *OAuth2Token) Create() (err error) {
	t.ID = CreateUUID()
	_, err = db.MustCols().UseBool().InsertOne(t)
	return
}

// Delete : データベースから削除します
func (t *OAuth2Token) Delete() (err error) {
	_, err = db.Delete(t)
	return
}

// GetOAuth2TokenByRefresh : リフレッシュトークンからOAuth2Tokenを取得します
func GetOAuth2TokenByRefresh(refresh string) (ot *OAuth2Token, err error) {
	ok, err := db.Where("refresh_token = ?", refresh).Get(ot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return
}

// GetOAuth2TokenByRefresh : 指定したユーザーIDのOAuth2Tokenを全て取得します
func GetOAuth2TokenByUser(userID string) (ts []*OAuth2Token, err error) {
	err = db.Where("user_id = ?", userID).Find(&ts)
	return
}

// GetOAuth2TokenByClient : 指定したクライアントのOAuth2Tokenを全て取得します
func GetOAuth2TokenByClient(clientID string) (ts []*OAuth2Token, err error) {
	err = db.Where("client_id = ?", clientID).Find(&ts)
	return
}

// GetOAuth2TokenByAccess : アクセストークンからOAuth2Tokenを取得します
func GetOAuth2TokenByAccess(access string) (ot *OAuth2Token, err error) {
	ok, err := db.Where("access_token = ?", access).Get(ot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return
}
