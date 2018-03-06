package model

import "time"

// Stamp スタンプ構造体
type Stamp struct {
	ID        string    `xorm:"char(36) pk"       json:"id"`
	CreatorID string    `xorm:"char(36) not null" json:"creatorId"`
	FileID    string    `xorm:"char(36) not null" json:"fileId"`
	IsDeleted bool      `xorm:"bool not null"     json:"-"`
	CreatedAt time.Time `xorm:"created"           json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated"           json:"updatedAt"`
}

// TableName : スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

func CreateStamp() (*Stamp, error) {

}

// DeleteStamp : 指定したIDのスタンプを削除します
func DeleteStamp(id string) error {
	var stamp Stamp
	ok, err := db.ID(id).Get(&stamp)
	if err != nil {
		return err
	}
	if ok {
		stamp.IsDeleted = true
		_, err = db.Update(&stamp)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllStamps : 全てのスタンプを取得します
func GetAllStamps() (stamps []*Stamp, err error) {
	err = db.Where("is_deleted = false").Find(&stamps)
	return
}
