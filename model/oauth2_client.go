package model

import (
	"github.com/jinzhu/gorm"
	"time"
)

// OAuth2Client OAuth2 クライアント構造体
type OAuth2Client struct {
	ID           string `gorm:"type:char(36);primary_key"`
	Name         string `gorm:"type:varchar(32)"`
	Description  string `gorm:"type:text"`
	Confidential bool
	CreatorID    string     `gorm:"type:char(36)"`
	Secret       string     `gorm:"type:varchar(36)"`
	RedirectURI  string     `gorm:"type:text"`
	Scopes       string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"precision:6"`
	UpdatedAt    time.Time  `gorm:"precision:6"`
	DeletedAt    *time.Time `gorm:"precision:6"`
}

// TableName OAuth2Clientのテーブル名
func (*OAuth2Client) TableName() string {
	return "oauth2_clients"
}

// Create データベースに挿入します
func (oc *OAuth2Client) Create() error {
	return db.Create(oc).Error
}

// Update データベースを更新します
func (oc *OAuth2Client) Update() error {
	return db.Model(oc).Updates(map[string]interface{}{
		"name":         oc.Name,
		"description":  oc.Description,
		"confidential": oc.Confidential,
		"creator_id":   oc.CreatorID,
		"secret":       oc.Secret,
		"redirect_uri": oc.RedirectURI,
		"scopes":       oc.Scopes,
	}).Error
}

// Delete クライアントを削除します
func (oc *OAuth2Client) Delete() (err error) {
	return db.Delete(oc).Error
}

// GetOAuth2ClientByClientID クライアントIDからOAuth2Clientを取得します
func GetOAuth2ClientByClientID(id string) (oc *OAuth2Client, err error) {
	oc = &OAuth2Client{}
	err = db.Where(OAuth2Client{ID: id}).Take(oc).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return
}

// GetOAuth2ClientsByUser 指定した登録者のOAuth2Clientを全て取得します
func GetOAuth2ClientsByUser(userID string) (cs []*OAuth2Client, err error) {
	err = db.Where(OAuth2Client{CreatorID: userID}).Find(&cs).Error
	return
}
