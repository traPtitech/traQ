package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// ClipFolder クリップフォルダの構造体
type ClipFolder struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key"                                            json:"id"`
	UserID    uuid.UUID `gorm:"type:char(36);unique_index:user_folder"                               json:"-"`
	Name      string    `gorm:"type:varchar(30);unique_index:user_folder" validate:"max=30,required" json:"name"`
	CreatedAt time.Time `gorm:"precision:6"                                                          json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6"                                                          json:"-"`
}

// TableName ClipFolderのテーブル名
func (*ClipFolder) TableName() string {
	return "clip_folders"
}

// Validate 構造体を検証します
func (f *ClipFolder) Validate() error {
	return validator.ValidateStruct(f)
}

// Clip clipの構造体
type Clip struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID `gorm:"type:char(36);unique_index:user_message"`
	MessageID uuid.UUID `gorm:"type:char(36);unique_index:user_message"`
	Message   Message   `gorm:"association_autoupdate:false;association_autocreate:false"`
	FolderID  uuid.UUID `gorm:"type:char(36)"`
	CreatedAt time.Time `gorm:"precision:6"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// TableName Clipのテーブル名
func (clip *Clip) TableName() string {
	return "clips"
}

// GetClipFolder 指定したIDのクリップフォルダを取得します
func GetClipFolder(id uuid.UUID) (*ClipFolder, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	f := &ClipFolder{}
	if err := db.Where(&ClipFolder{ID: id}).Take(f).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// GetClipFolders 指定したユーザーのクリップフォルダを全て取得します
func GetClipFolders(userID uuid.UUID) (res []*ClipFolder, err error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	res = make([]*ClipFolder, 0)
	err = db.Where(&ClipFolder{UserID: userID}).Order("name").Find(&res).Error
	return
}

// CreateClipFolder クリップフォルダを作成します
func CreateClipFolder(userID uuid.UUID, name string) (*ClipFolder, error) {
	f := &ClipFolder{
		ID:     uuid.NewV4(),
		UserID: userID,
		Name:   name,
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := db.Create(f).Error; err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateClipFolderName クリップフォルダ名を更新します
func UpdateClipFolderName(id uuid.UUID, name string) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return db.Where(&ClipFolder{ID: id}).Update("name", name).Error
}

// DeleteClipFolder クリップフォルダを削除します
func DeleteClipFolder(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return db.Delete(&ClipFolder{ID: id}).Error
}

// GetClipMessage 指定したIDのクリップを取得します
func GetClipMessage(id uuid.UUID) (*Clip, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	c := &Clip{}
	if err := db.Preload("Message").Where(&Clip{ID: id}).Take(c).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// GetClipMessages 指定したフォルダのクリップを全て取得します
func GetClipMessages(folderID uuid.UUID) (res []*Clip, err error) {
	if folderID == uuid.Nil {
		return nil, ErrNilID
	}
	res = make([]*Clip, 0)
	err = db.Preload("Message").Where(&Clip{FolderID: folderID}).Order("updated_at").Find(&res).Error
	return
}

// GetClipMessagesByUser 指定したユーザーのクリップを全て取得します
func GetClipMessagesByUser(userID uuid.UUID) (res []*Clip, err error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	res = make([]*Clip, 0)
	err = db.Preload("Message").Where(&Clip{UserID: userID}).Order("updated_at").Find(&res).Error
	return
}

// CreateClip クリップを作成します
func CreateClip(messageID, folderID, userID uuid.UUID) (*Clip, error) {
	c := &Clip{
		ID:        uuid.NewV4(),
		UserID:    userID,
		MessageID: messageID,
		FolderID:  folderID,
	}
	if err := db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// ChangeClipFolder クリップのフォルダを変更します
func ChangeClipFolder(clipID, folderID uuid.UUID) error {
	if clipID == uuid.Nil || folderID == uuid.Nil {
		return ErrNilID
	}
	return db.Where(&Clip{ID: clipID}).Updates(&Clip{FolderID: folderID}).Error
}

// DeleteClip クリップを削除します
func DeleteClip(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	return db.Delete(&Clip{ID: id}).Error
}
