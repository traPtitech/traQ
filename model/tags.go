package model

import "github.com/traPtitech/traQ/utils/validator"

// Tag tag_idの管理をする構造体
type Tag struct {
	ID   string `xorm:"char(36) pk"                 validate:"uuid,required"`
	Name string `xorm:"varchar(30) not null unique" validate:"required,max=30"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// Validate 構造体を検証します
func (t *Tag) Validate() error {
	return validator.ValidateStruct(t)
}

// Create DBにタグを追加
func (t *Tag) Create() (err error) {
	t.ID = CreateUUID()
	if err = t.Validate(); err != nil {
		return err
	}

	_, err = db.InsertOne(t)
	return
}

// Exists DBにその名前のタグが存在するかを確認
func (t *Tag) Exists() (bool, error) {
	if t.Name == "" {
		return false, ErrInvalidParam
	}
	return db.Get(t)
}

// GetTagByID 引数のIDを持つTag構造体を返す
func GetTagByID(ID string) (*Tag, error) {
	tag := &Tag{
		ID: ID,
	}

	has, err := db.Get(tag)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrNotFound
	}
	return tag, nil
}

// GetAllTags 全てのタグを取得する
func GetAllTags() (result []*Tag, err error) {
	err = db.Find(&result)
	return
}
