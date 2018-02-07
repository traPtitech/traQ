package model

import (
	"fmt"
)

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    string `xorm:"char(36) pk"`
	TagID     string `xorm:"char(36) pk"`
	IsLocked  bool   `xorm:"bool not null"`
	CreatedAt string `xorm:"created not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// TableName DBの名前を指定
func (userTag *UsersTag) TableName() string {
	return "users_tags"
}

// Create DBに新規タグを追加します
func (userTag *UsersTag) Create(name string) error {
	if userTag.UserID == "" {
		return fmt.Errorf("UserID is empty")
	}

	tag := &Tag{
		Name: name,
	}
	has, err := tag.Exists()
	if err != nil {
		return fmt.Errorf("Failed to check whether the tag exist: %v", err)
	}
	if !has {
		if err := tag.Create(); err != nil {
			return err
		}
	}

	userTag.TagID = tag.ID
	userTag.IsLocked = false
	if _, err := db.Insert(userTag); err != nil {
		return fmt.Errorf("Failed to create tag object: %v", err)
	}
	return nil
}

// Update データの更新をします
func (userTag *UsersTag) Update() error {
	if _, err := db.Where("user_id = ? AND tag_id = ?", userTag.UserID, userTag.TagID).UseBool().Update(userTag); err != nil {
		return fmt.Errorf("Failed to update tag: %v", err)
	}
	return nil
}

// Delete データを消去します。正しく消せた場合はレシーバはnilになります
func (userTag *UsersTag) Delete() error {
	if _, err := db.Delete(userTag); err != nil {
		return fmt.Errorf("Failed to delete tag: %v", err)
	}
	return nil
}

// GetUserTagsByUserID userIDに紐づくtagのリストを返します
func GetUserTagsByUserID(userID string) ([]*UsersTag, error) {
	var tags []*UsersTag
	if err := db.Where("user_id = ?", userID).Asc("created_at").Find(&tags); err != nil {
		return nil, fmt.Errorf("Failed to find tags: %v", err)
	}
	return tags, nil
}

// GetTag userIDとtagIDで一意に定まるタグを返します
func GetTag(userID, tagID string) (*UsersTag, error) {
	var tag UsersTag
	has, err := db.Where("user_id = ? AND tag_id = ?", userID, tagID).Get(&tag)
	if err != nil {
		return nil, fmt.Errorf("Failed to find tag: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("this tag doesn't exist. tagID = %s", tagID)
	}
	return &tag, nil
}
