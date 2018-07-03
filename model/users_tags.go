package model

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/utils/validator"
	"time"

	"github.com/satori/go.uuid"
)

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    string `gorm:"type:char(36);primary_key"      validate:"uuid,required"`
	TagID     string `gorm:"type:char(36);primary_key"      validate:"uuid,required"`
	Tag       Tag    `gorm:"association_autoupdate:false;association_autocreate:false"`
	IsLocked  bool
	CreatedAt time.Time `gorm:"precision:6;index"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*UsersTag) TableName() string {
	return "users_tags"
}

// Validate 構造体を検証します
func (ut *UsersTag) Validate() error {
	return validator.ValidateStruct(ut)
}

// AddUserTag ユーザーにタグを付与します
func AddUserTag(userID, tagID uuid.UUID) error {
	ut := &UsersTag{
		UserID: userID.String(),
		TagID:  tagID.String(),
	}
	return db.Create(ut).Error
}

// ChangeUserTagLock ユーザーのタグのロック状態を変更します
func ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	return db.Where(UsersTag{UserID: userID.String(), TagID: tagID.String()}).Update("is_locked", locked).Error
}

// DeleteUserTag ユーザーからタグを削除します
func DeleteUserTag(userID, tagID uuid.UUID) error {
	return db.Where(UsersTag{UserID: userID.String(), TagID: tagID.String()}).Delete(UsersTag{}).Error
}

// GetUserTagsByUserID userIDに紐づくtagのリストを返します
func GetUserTagsByUserID(userID uuid.UUID) (tags []*UsersTag, err error) {
	err = db.Preload("Tag").Where("user_id = ?", userID.String()).Order("created_at").Find(&tags).Error
	return
}

// GetUserTag userIDとtagIDで一意に定まるタグを返します
func GetUserTag(userID, tagID uuid.UUID) (*UsersTag, error) {
	ut := &UsersTag{}
	if err := db.Preload("Tag").Where(UsersTag{UserID: userID.String(), TagID: tagID.String()}).Take(ut).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ut, nil
}

// GetUserIDsByTag 指定したタグを持った全ユーザーのUUIDを返します
func GetUserIDsByTag(tag string) ([]uuid.UUID, error) {
	var arr []string
	err := db.
		Model(UsersTag{}).
		Joins("INNER JOIN tags ON users_tags.tag_id = tags.id AND tags.name = ?", tag).
		Pluck("users_tags.user_id", &arr).
		Error
	if err != nil {
		return nil, err
	}

	return convertStringSliceToUUIDSlice(arr), nil
}

// GetUsersByTag 指定したタグを持った全ユーザーを取得します
func GetUsersByTag(tag string) (arr []*User, err error) {
	err = db.
		Where("id IN ?", db.
			Model(UsersTag{}).
			Select("users_tags.user_id").
			Joins("INNER JOIN tags ON users_tags.tag_id = tags.id AND tags.name = ?", tag).
			QueryExpr()).
		Find(&arr).
		Error
	return
}

// GetUserIDsByTagID 指定したタグIDのタグを持った全ユーザーのIDを返します
func GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error) {
	var arr []string
	err := db.
		Model(UsersTag{}).
		Where("tag_id = ?", tagID.String()).
		Pluck("user_id", &arr).
		Error
	if err != nil {
		return nil, err
	}
	return convertStringSliceToUUIDSlice(arr), nil
}
