package model

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

var (
	// ErrUserAlreadyHasTag 対象のユーザーは既に対象のタグを持っています
	ErrUserAlreadyHasTag = errors.New("the user already has the tag")
)

// Tag tag_idの管理をする構造体
type Tag struct {
	ID         uuid.UUID `gorm:"type:char(36);primary_key"`
	Name       string    `gorm:"type:varchar(30);unique"   validate:"required,max=30"`
	Restricted bool
	Type       string    `gorm:"type:varchar(30)"`
	CreatedAt  time.Time `gorm:"precision:6"`
	UpdatedAt  time.Time `gorm:"precision:6"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// Validate 構造体を検証します
func (t *Tag) Validate() error {
	return validator.ValidateStruct(t)
}

// UsersTag userTagの構造体
type UsersTag struct {
	UserID    uuid.UUID `gorm:"type:char(36);primary_key"`
	TagID     uuid.UUID `gorm:"type:char(36);primary_key"`
	Tag       Tag       `gorm:"association_autoupdate:false;association_autocreate:false"`
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

// CreateTag タグを作成します
func CreateTag(name string, restricted bool, tagType string) (*Tag, error) {
	t := &Tag{
		ID:         uuid.NewV4(),
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, db.Create(t).Error
}

// ChangeTagType タグの種類を変更します
func ChangeTagType(id uuid.UUID, tagType string) error {
	return db.Model(Tag{ID: id}).Update("type", tagType).Error
}

// ChangeTagRestrict タグの制限を変更します
func ChangeTagRestrict(id uuid.UUID, restrict bool) error {
	return db.Model(Tag{ID: id}).Update("restricted", restrict).Error
}

// GetTagByID 引数のIDを持つTag構造体を返す
func GetTagByID(id uuid.UUID) (*Tag, error) {
	tag := &Tag{}
	if err := db.Where(Tag{ID: id}).Take(tag).Error; err != nil {
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

// GetOrCreateTagByName 引数のタグを取得するか、生成したものを返します。
func GetOrCreateTagByName(name string) (*Tag, error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	tag := &Tag{}
	if err := db.Where(Tag{Name: name}).Take(tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			tag.Name = name
			if err = db.Create(tag).Error; err == nil {
				return tag, nil
			}
		}
		return nil, err
	}
	return tag, nil
}

// AddUserTag ユーザーにタグを付与します
func AddUserTag(userID, tagID uuid.UUID) error {
	ut := &UsersTag{
		UserID: userID,
		TagID:  tagID,
	}
	if err := db.Create(ut).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return ErrUserAlreadyHasTag
		}
		return err
	}
	return nil
}

// ChangeUserTagLock ユーザーのタグのロック状態を変更します
func ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	return db.Model(UsersTag{}).Where(UsersTag{UserID: userID, TagID: tagID}).Update("is_locked", locked).Error
}

// DeleteUserTag ユーザーからタグを削除します
func DeleteUserTag(userID, tagID uuid.UUID) error {
	return db.Where(UsersTag{UserID: userID, TagID: tagID}).Delete(UsersTag{}).Error
}

// GetUserTagsByUserID userIDに紐づくtagのリストを返します
func GetUserTagsByUserID(userID uuid.UUID) (tags []*UsersTag, err error) {
	err = db.Preload("Tag").Where("user_id = ?", userID.String()).Order("created_at").Find(&tags).Error
	return
}

// GetUserTag userIDとtagIDで一意に定まるタグを返します
func GetUserTag(userID, tagID uuid.UUID) (*UsersTag, error) {
	ut := &UsersTag{}
	if err := db.Preload("Tag").Where(UsersTag{UserID: userID, TagID: tagID}).Take(ut).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ut, nil
}

// GetUserIDsByTag 指定したタグを持った全ユーザーのUUIDを返します
func GetUserIDsByTag(tag string) (arr []uuid.UUID, err error) {
	err = db.
		Model(UsersTag{}).
		Joins("INNER JOIN tags ON users_tags.tag_id = tags.id AND tags.name = ?", tag).
		Pluck("users_tags.user_id", &arr).
		Error
	return arr, err
}

// GetUsersByTag 指定したタグを持った全ユーザーを取得します
func GetUsersByTag(tag string) (arr []*User, err error) {
	err = db.
		Where("id IN (?)", db.
			Model(UsersTag{}).
			Select("users_tags.user_id").
			Joins("INNER JOIN tags ON users_tags.tag_id = tags.id AND tags.name = ?", tag).
			QueryExpr()).
		Find(&arr).
		Error
	return
}

// GetUserIDsByTagID 指定したタグIDのタグを持った全ユーザーのIDを返します
func GetUserIDsByTagID(tagID uuid.UUID) (arr []uuid.UUID, err error) {
	err = db.
		Model(UsersTag{}).
		Where("tag_id = ?", tagID.String()).
		Pluck("user_id", &arr).
		Error
	return arr, err
}
