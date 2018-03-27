package model

import (
	"errors"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

var (
	//ErrStampInvalidName : スタンプ名が不正です
	ErrStampInvalidName = errors.New("invalid name")
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        string    `xorm:"char(36) pk"                 json:"id"        validate:"uuid,required"`
	Name      string    `xorm:"varchar(32) not null unique" json:"name"      validate:"name,required"`
	CreatorID string    `xorm:"char(36) not null"           json:"creatorId" validate:"uuid,required"`
	FileID    string    `xorm:"char(36) not null"           json:"fileId"    validate:"uuid,required"`
	IsDeleted bool      `xorm:"bool not null"               json:"-"`
	CreatedAt time.Time `xorm:"created"                     json:"createdAt"`
	UpdatedAt time.Time `xorm:"updated"                     json:"updatedAt"`
}

// TableName : スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// Validate 構造体を検証します
func (s *Stamp) Validate() error {
	return validator.ValidateStruct(s)
}

// Update : スタンプを修正します
func (s *Stamp) Update() (err error) {
	if err = s.Validate(); err != nil {
		return
	}

	_, err = db.ID(s.ID).Update(s)
	return
}

// CreateStamp : スタンプを作成します
func CreateStamp(name, fileID, userID string) (*Stamp, error) {
	stamp := &Stamp{
		ID:        CreateUUID(),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
		IsDeleted: false,
	}

	if err := stamp.Validate(); err != nil {
		return nil, err
	}

	if _, err := db.InsertOne(stamp); err != nil {
		return nil, err
	}

	return stamp, nil
}

// GetStamp : 指定したIDのスタンプを取得します
func GetStamp(id string) (*Stamp, error) {
	var stamp Stamp
	ok, err := db.ID(id).Where("is_deleted = false").Get(&stamp)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
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
		_, err = db.ID(stamp.ID).Update(&stamp)
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
