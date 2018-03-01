package model

import (
	"fmt"
)

// Tag tag_idの管理をする構造体
type Tag struct {
	ID   string `xorm:"char(36) pk"`
	Name string `xorm:"varchar(30) not null unique"`
}

// TableName DBの名前を指定
func (tag *Tag) TableName() string {
	return "tags"
}

// Create DBにタグを追加
func (tag *Tag) Create() error {
	if tag.Name == "" {
		return fmt.Errorf("Name is empty")
	}
	tag.ID = CreateUUID()

	if _, err := db.Insert(tag); err != nil {
		return fmt.Errorf("Failed to create tags object: %v", err)
	}
	return nil
}

// Exists DBにその名前のタグが存在するかを確認
func (tag *Tag) Exists() (bool, error) {
	if tag.Name == "" {
		return false, fmt.Errorf("Name is empty")
	}
	return db.Get(tag)
}

// GetTagByID 引数のIDを持つTag構造体を返す
func GetTagByID(ID string) (*Tag, error) {
	tag := &Tag{
		ID: ID,
	}

	has, err := db.Get(tag)
	if err != nil {
		return nil, fmt.Errorf("Failed to get tag: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("tag has this ID is not found: ID = %s", ID)
	}
	return tag, nil
}

// GetAllTags 全てのタグを取得する
func GetAllTags() (result []*Tag, err error) {
	err = db.Find(&result)
	return
}
