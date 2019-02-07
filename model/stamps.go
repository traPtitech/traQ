package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Stamp スタンプ構造体
type Stamp struct {
	ID        uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	Name      string     `gorm:"type:varchar(32);unique"   json:"name"      validate:"name,required"`
	CreatorID uuid.UUID  `gorm:"type:char(36)"             json:"creatorId"`
	FileID    uuid.UUID  `gorm:"type:char(36)"             json:"fileId"`
	CreatedAt time.Time  `gorm:"precision:6"               json:"createdAt"`
	UpdatedAt time.Time  `gorm:"precision:6"               json:"updatedAt"`
	DeletedAt *time.Time `gorm:"precision:6"               json:"-"`
}

// TableName スタンプテーブル名を取得します
func (*Stamp) TableName() string {
	return "stamps"
}

// Validate 構造体を検証します
func (s *Stamp) Validate() error {
	return validator.ValidateStruct(s)
}

// UpdateStamp スタンプを更新します
func UpdateStamp(stampID uuid.UUID, s Stamp) error {
	if stampID == uuid.Nil {
		return ErrNilID
	}
	s.ID = uuid.Nil
	s.CreatedAt = time.Time{}
	s.UpdatedAt = time.Time{}
	s.DeletedAt = nil
	if err := validator.ValidateVar(s.Name, "name"); err != nil {
		return err
	}

	return db.Where(&Stamp{ID: stampID}).Updates(&s).Error
}

// CreateStamp スタンプを作成します
func CreateStamp(name string, fileID, userID uuid.UUID) (*Stamp, error) {
	if fileID == uuid.Nil {
		return nil, ErrNilID
	}

	stamp := &Stamp{
		ID:        uuid.NewV4(),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
	}
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	if err := db.Create(stamp).Error; err != nil {
		return nil, err
	}

	return stamp, nil
}

// GetStamp 指定したIDのスタンプを取得します
func GetStamp(id uuid.UUID) (*Stamp, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	s := &Stamp{}
	if err := db.Where(&Stamp{ID: id}).Take(s).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

// DeleteStamp 指定したIDのスタンプを削除します
func DeleteStamp(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return db.Delete(&Stamp{ID: id}).Error
}

// GetAllStamps 全てのスタンプを取得します
func GetAllStamps() (stamps []*Stamp, err error) {
	err = db.Find(&stamps).Error
	return
}

// StampExists 指定したIDのスタンプが存在するかどうか
func StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	c := 0
	if err := db.Model(Stamp{}).Where(&Stamp{ID: id}).Count(&c).Error; err != nil {
		return false, err
	}
	return c > 0, nil
}

// IsStampNameDuplicate 指定した名前のスタンプが存在するかどうか
func IsStampNameDuplicate(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	c := 0
	if err := db.Model(Stamp{}).Where(&Stamp{Name: name}).Count(&c).Error; err != nil {
		return false, err
	}
	return c > 0, nil
}
