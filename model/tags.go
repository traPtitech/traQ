package model

import (
	"fmt"
)

// Tag userTagの構造体
type Tag struct {
	ID        string `xorm:"char(36) pk"`
	UserID    string `xorm:"char(36) not null"`
	Tag       string `xorm:"varcher(30) not null"`
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

	tag.ID = CreateUUID()
	tag.IsLocked = false
	if _, err := db.Insert(tag); err != nil {
		return fmt.Errorf("Failed to create message object: %v", err)
	}
	return nil
}

// Update データの更新をします
func (tag *Tag) Update() error {
	if _, err := db.ID(tag.ID).UseBool().Update(tag); err != nil {
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

// GetTagsByUserID userIDに紐づくtagのリストを返します
func GetTagsByUserID(userID string) ([]*Tag, error) {
	var tags []*Tag
	if err := db.Where("user_id = ?", userID).Asc("created_at").Find(&tags); err != nil {
		return nil, fmt.Errorf("Failed to find tags: %v", err)
	}
	return tags, nil
}

// GetTag userIDとtagで一意に定まるタグを返します
func GetTag(tagID string) (*Tag, error) {
	var tag Tag
	has, err := db.ID(tagID).Get(&tag)
	if err != nil {
		return nil, fmt.Errorf("Failed to find tag: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("this tag doesn't exist. tagID = %s", tagID)
	}
	return &tag, nil
}
