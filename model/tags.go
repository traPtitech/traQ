package model

import (
	"fmt"

	"github.com/go-xorm/core"
)

// Tag userTagの構造体
type Tag struct {
	UserID    string `xorm:"char(36) pk"`
	Tag       string `xorm:"varcher(30) pk"`
	IsLocked  bool   `xorm:"bool not null"`
	CreatedAt string `xorm:"created not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// TableName DBの名前を指定
func (tag *Tag) TableName() string {
	return "users_tags"
}

// Create DBに新規タグを追加します
func (tag *Tag) Create() error {
	if tag.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}
	if tag.Tag == "" {
		return fmt.Errorf("Tag is empty")
	}

	tag.IsLocked = false
	if _, err := db.Insert(tag); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

// Update データの更新をします
func (tag *Tag) Update() error {
	if _, err := db.UseBool().Update(tag, &Tag{UserID: tag.UserID, Tag: tag.Tag}); err != nil {
		return fmt.Errorf("Failed to update tag: %v", err)
	}
	return nil
}

// Delete データを消去します。正しく消せた場合はレシーバはnilになります
func (tag *Tag) Delete() error {
	if _, err := db.Delete(tag); err != nil {
		return fmt.Errorf("Failed to delete tag: %v", err)
	}
	return nil
}

// GetTagsByID userIDに紐づくtagのリストを返します
func GetTagsByID(userID string) ([]*Tag, error) {
	var tags []*Tag
	if err := db.Where("user_id = ?", userID).Asc("created_at").Find(&tags); err != nil {
		return nil, fmt.Errorf("Failed to find tags: %v", err)
	}
	return tags, nil
}

// GetTag userIDとtagで一意に定まるタグを返します
func GetTag(userID, tagText string) (*Tag, error) {
	var tag Tag
	has, err := db.ID(core.PK{userID, tagText}).Get(&tag)
	if err != nil {
		return nil, fmt.Errorf("Failed to find tag: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("this tag doesn't exist. userID = %s, Tag = %s", userID, tagText)
	}
	return &tag, nil
}
