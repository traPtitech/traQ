package model

import (
	"errors"
	"regexp"
	"time"
)

var (
	stampNameRegexp = regexp.MustCompile("[a-zA-Z0-9+_-]{1,32}")

	//ErrStampInvalidName : スタンプ名が不正です
	ErrStampInvalidName = errors.New("invalid name")
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        string    `xorm:"char(36) pk"                 json:"id"`
	Name      string    `xorm:"varchar(32) not null unique" json:"name"`
	CreatorID string    `xorm:"char(36) not null"           json:"creatorId"`
	FileID    string    `xorm:"char(36) not null"           json:"fileId"`
	IsDeleted bool      `xorm:"bool not null"               json:"-"`
	CreatedAt time.Time `xorm:"created"                     json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated"                     json:"updatedAt"`
}

// TableName : スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// Update : スタンプを修正します
func (s *Stamp) Update() error {
	if !stampNameRegexp.MatchString(s.Name) {
		return ErrStampInvalidName
	}

	if _, err := db.ID(s.ID).Update(s); err != nil {
		return err
	}

	return nil
}

// CreateStamp : スタンプを作成します
func CreateStamp(name, fileID, userID string) (*Stamp, error) {
	if !stampNameRegexp.MatchString(name) {
		return nil, ErrStampInvalidName
	}

	stamp := &Stamp{
		ID:        CreateUUID(),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
		IsDeleted: false,
	}
	if _, err := db.InsertOne(stamp); err != nil {
		return nil, err
	}

	return stamp, nil
}

// GetStamp : 指定したIDのスタンプを取得します
func GetStamp(id string) (*Stamp, error) {
	var stamp Stamp
	ok, err := db.ID(id).Get(&stamp)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &stamp, nil
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
