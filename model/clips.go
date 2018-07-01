package model

import (
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// ClipFolder クリップフォルダの構造体
type ClipFolder struct {
	ID        string    `gorm:"type:char(36);primary_key"              validate:"uuid,required"`
	UserID    string    `gorm:"type:char(36);unique_index:user_folder" validate:"uuid,required"`
	Name      string    `gorm:"size:30;unique_index:user_folder"       validate:"max=30,required"`
	CreatedAt time.Time `gorm:"precision:6"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// GetID IDをuuid.UUIDとして取得します
func (f *ClipFolder) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(f.ID))
}

// GetUID UserIDをuuid.UUIDとして取得します
func (f *ClipFolder) GetUID() uuid.UUID {
	return uuid.Must(uuid.FromString(f.UserID))
}

// TableName ClipFolderのテーブル名
func (*ClipFolder) TableName() string {
	return "clip_folders"
}

// BeforeCreate db.Create時に自動的に呼ばれます
func (f *ClipFolder) BeforeCreate(scope *gorm.Scope) error {
	f.ID = CreateUUID()
	return f.Validate()
}

// Validate 構造体を検証します
func (f *ClipFolder) Validate() error {
	return validator.ValidateStruct(f)
}

// Clip clipの構造体
type Clip struct {
	ID        string    `gorm:"type:char(36);primary_key"               validate:"uuid,required"`
	UserID    string    `gorm:"type:char(36);unique_index:user_message" validate:"uuid,required"`
	MessageID string    `gorm:"type:char(36);unique_index:user_message" validate:"uuid,required"`
	Message   Message   `gorm:"association_autoupdate:false;association_autocreate:false"`
	FolderID  string    `gorm:"type:char(36)"                           validate:"uuid,required"`
	CreatedAt time.Time `gorm:"precision:6"`
	UpdatedAt time.Time `gorm:"precision:6"`
}

// GetID IDをuuid.UUIDとして取得します
func (clip *Clip) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(clip.ID))
}

// GetUID UserIDをuuid.UUIDとして取得します
func (clip *Clip) GetUID() uuid.UUID {
	return uuid.Must(uuid.FromString(clip.UserID))
}

// GetMID MessageIDをIDをuuid.UUIDとして取得します
func (clip *Clip) GetMID() uuid.UUID {
	return uuid.Must(uuid.FromString(clip.MessageID))
}

// GetFID FolderIDをuuid.UUIDとして取得します
func (clip *Clip) GetFID() uuid.UUID {
	return uuid.Must(uuid.FromString(clip.FolderID))
}

// TableName Clipのテーブル名
func (clip *Clip) TableName() string {
	return "clips"
}

// BeforeCreate db.Create時に自動的に呼ばれます
func (clip *Clip) BeforeCreate(scope *gorm.Scope) error {
	clip.ID = CreateUUID()
	return clip.Validate()
}

// Validate 構造体を検証します
func (clip *Clip) Validate() error {
	return validator.ValidateStruct(clip)
}

// GetClipFolder 指定したIDのクリップフォルダを取得します
func GetClipFolder(id uuid.UUID) (*ClipFolder, error) {
	f := &ClipFolder{}
	if err := db.Where(ClipFolder{ID: id.String()}).Take(f).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// GetClipFolders 指定したユーザーのクリップフォルダを全て取得します
func GetClipFolders(userID uuid.UUID) (res []*ClipFolder, err error) {
	err = db.Where(ClipFolder{UserID: userID.String()}).Order("name").Find(&res).Error
	return
}

// CreateClipFolder クリップフォルダを作成します
func CreateClipFolder(userID uuid.UUID, name string) (*ClipFolder, error) {
	f := &ClipFolder{
		UserID: userID.String(),
		Name:   name,
	}
	err := db.Create(f).Error
	if err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateClipFolderName クリップフォルダ名を更新します
func UpdateClipFolderName(id uuid.UUID, name string) error {
	return db.Where(ClipFolder{ID: id.String()}).Update("name", name).Error
}

// DeleteClipFolder クリップフォルダを削除します
func DeleteClipFolder(id uuid.UUID) error {
	return db.Delete(ClipFolder{ID: id.String()}).Error
}

// GetClipMessage 指定したIDのクリップを取得します
func GetClipMessage(id uuid.UUID) (*Clip, error) {
	c := &Clip{}
	if err := db.Preload("Message").Where(Clip{ID: id.String()}).Take(c).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// GetClipMessages 指定したフォルダのクリップを全て取得します
func GetClipMessages(folderID uuid.UUID) (res []*Clip, err error) {
	err = db.Preload("Message").Where(Clip{FolderID: folderID.String()}).Order("updated_at").Find(&res).Error
	return
}

// GetClipMessagesByUser 指定したユーザーのクリップを全て取得します
func GetClipMessagesByUser(userID uuid.UUID) (res []*Clip, err error) {
	err = db.Preload("Message").Where(Clip{UserID: userID.String()}).Order("updated_at").Find(&res).Error
	return
}

// CreateClip クリップを作成します
func CreateClip(messageID, folderID, userID uuid.UUID) (*Clip, error) {
	c := &Clip{
		UserID:    userID.String(),
		MessageID: messageID.String(),
		FolderID:  folderID.String(),
	}
	if err := db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// ChangeClipFolder クリップのフォルダを変更します
func ChangeClipFolder(clipID, folderID uuid.UUID) error {
	return db.Where(Clip{ID: clipID.String()}).Updates(Clip{FolderID: folderID.String()}).Error
}

// DeleteClip クリップを削除します
func DeleteClip(id uuid.UUID) error {
	return db.Delete(Clip{ID: id.String()}).Error
}
