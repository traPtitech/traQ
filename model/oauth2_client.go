package model

import "time"

// OAuth2Client : OAuth2 クライアント構造体
type OAuth2Client struct {
	ID           string    `xorm:"char(36) not null pk"`
	Name         string    `xorm:"varchar(32) not null"`
	Description  string    `xorm:"text not null"`
	Confidential bool      `xorm:"bool not null"`
	CreatorID    string    `xorm:"char(36) not null"`
	Secret       string    `xorm:"varchar(36) not null"`
	RedirectURI  string    `xorm:"text not null"`
	Scopes       string    `xorm:"text not null"`
	IsDeleted    bool      `xorm:"bool not null"`
	CreatedAt    time.Time `xorm:"created"`
	UpdatedAt    time.Time `xorm:"updated"`
}

// TableName : OAuth2Clientのテーブル名
func (*OAuth2Client) TableName() string {
	return "oauth2_clients"
}

// Create : データベースに挿入します
func (oc *OAuth2Client) Create() (err error) {
	oc.ID = CreateUUID()
	_, err = db.MustCols().UseBool().InsertOne(oc)
	return
}

// Update : データベースを更新します
func (oc *OAuth2Client) Update() (err error) {
	_, err = db.ID(oc.ID).MustCols().UseBool().Update(oc)
	return
}

// Delete : クライアントを削除します
func (oc *OAuth2Client) Delete() (err error) {
	oc.IsDeleted = true
	_, err = db.ID(oc.ID).MustCols().UseBool().Update(oc)
	return
}

// GetOAuth2ClientByClientID : クライアントIDからOAuth2Clientを取得します
func GetOAuth2ClientByClientID(id string) (oc *OAuth2Client, err error) {
	ok, err := db.ID(id).Where("is_deleted = false").Get(oc)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return
}

// GetOAuth2ClientsByUser : 指定した登録者のOAuth2Clientを全て取得します
func GetOAuth2ClientsByUser(userID string) (cs []*OAuth2Client, err error) {
	err = db.Where("is_deleted = false AND creator_id = ?", userID).Find(&cs)
	return
}
