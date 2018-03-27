package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/validator"
	"time"

	"github.com/go-xorm/builder"
	"github.com/satori/go.uuid"
)

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    string    `xorm:"char(36) pk"      validate:"uuid,required"`
	TagID     string    `xorm:"char(36) pk"      validate:"uuid,required"`
	IsLocked  bool      `xorm:"bool not null"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

// TableName DBの名前を指定
func (*UsersTag) TableName() string {
	return "users_tags"
}

// Validate 構造体を検証します
func (ut *UsersTag) Validate() error {
	return validator.ValidateStruct(ut)
}

// Create DBに新規タグを追加します
func (ut *UsersTag) Create(name string) error {
	if ut.UserID == "" {
		return ErrInvalidParam
	}

	t := &Tag{
		Name: name,
	}
	has, err := t.Exists()
	if err != nil {
		return err
	}
	if !has {
		if err := t.Create(); err != nil {
			return err
		}
	}

	ut.TagID = t.ID
	ut.IsLocked = false
	if _, err := db.Insert(ut); err != nil {
		return err
	}
	return nil
}

// Update データの更新をします
func (ut *UsersTag) Update() (err error) {
	_, err = db.Where("user_id = ? AND tag_id = ?", ut.UserID, ut.TagID).UseBool().Update(ut)
	return
}

// Delete データを消去します。正しく消せた場合はレシーバはnilになります
func (ut *UsersTag) Delete() (err error) {
	_, err = db.Delete(ut)
	return
}

// GetUserTagsByUserID userIDに紐づくtagのリストを返します
func GetUserTagsByUserID(userID string) (tags []*UsersTag, err error) {
	err = db.Where("user_id = ?", userID).Asc("created_at").Find(&tags)
	return
}

// GetTag userIDとtagIDで一意に定まるタグを返します
func GetTag(userID, tagID string) (*UsersTag, error) {
	if _, err := GetUser(userID); err != nil {
		return nil, err
	}

	if _, err := GetTagByID(tagID); err != nil {
		return nil, err
	}

	var ut = &UsersTag{}
	has, err := db.Where("user_id = ? AND tag_id = ?", userID, tagID).Get(ut)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}
	return ut, nil
}

// GetUserIDsByTags 指定したタグを持った全ユーザーのUUIDを返します
func GetUserIDsByTags(tags []string) ([]uuid.UUID, error) {
	var arr []string

	if err := db.Table(&UsersTag{}).Join("INNER", "tags", "users_tags.tag_id = tags.id").Where(builder.In("tags.name", tags)).Cols("user_id").Find(&arr); err != nil {
		return nil, fmt.Errorf("failed to get user ids by tag: %v", err)
	}

	result := make([]uuid.UUID, len(arr))
	for i, v := range arr {
		result[i] = uuid.FromStringOrNil(v)
	}

	return result, nil
}
