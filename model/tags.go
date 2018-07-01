package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Tag tag_idの管理をする構造体
type Tag struct {
	ID         string `gorm:"type:char(36);primary_key"`
	Name       string `gorm:"size:30;unique"            validate:"required,max=30"`
	Restricted bool
	Type       string
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// GetID タグのUUIDを返します
func (t *Tag) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(t.ID))
}

// BeforeCreate db.Create時に自動的に呼ばれます
func (t *Tag) BeforeCreate(scope *gorm.Scope) error {
	t.ID = CreateUUID()
	return t.Validate()
}

// Validate 構造体を検証します
func (t *Tag) Validate() error {
	return validator.ValidateStruct(t)
}

// CreateTag タグを作成します
func CreateTag(name string, restricted bool, tagType string) (*Tag, error) {
	t := &Tag{
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
	}
	if err := db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

// ChangeTagType タグの種類を変更します
func ChangeTagType(id uuid.UUID, tagType string, restricted bool) (err error) {
	err = db.Model(Tag{ID: id.String()}).Updates(map[string]interface{}{"restricted": restricted, "type": tagType}).Error
	return
}

// GetTagByID 引数のIDを持つTag構造体を返す
func GetTagByID(id uuid.UUID) (*Tag, error) {
	tag := &Tag{}
	if err := db.Where(Tag{ID: id.String()}).Take(tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return tag, nil
}

// GetTagByName 引数のタグのTag構造体を返す
func GetTagByName(name string) (*Tag, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	tag := &Tag{}
	if err := db.Where(Tag{Name: name}).Take(tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return tag, nil
}

// GetAllTags 全てのタグを取得する
func GetAllTags() (result []*Tag, err error) {
	err = db.Find(&result).Error
	return
}
