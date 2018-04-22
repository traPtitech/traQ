package model

import (
	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// ClipFolder クリップフォルダの構造体
type ClipFolder struct {
	ID        string    `xorm:"char(36) pk"                              validate:"uuid,required"`
	UserID    string    `xorm:"char(36) not null unique(user_folder)"    validate:"uuid,required"`
	Name      string    `xorm:"varchar(30) not null unique(user_folder)" validate:"max=30,required"`
	UpdatedAt time.Time `xorm:"updated"`
	CreatedAt time.Time `xorm:"created"`
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

// Validate 構造体を検証します
func (f *ClipFolder) Validate() error {
	return validator.ValidateStruct(f)
}

// Clip clipの構造体
type Clip struct {
	ID        string    `xorm:"char(36) pk"                            validate:"uuid,required"`
	UserID    string    `xorm:"char(36) not null unique(user_message)" validate:"uuid,required"`
	MessageID string    `xorm:"char(36) not null unique(user_message)" validate:"uuid,required"`
	FolderID  string    `xorm:"char(36) not null"                      validate:"uuid,required"`
	UpdatedAt time.Time `xorm:"updated"`
	CreatedAt time.Time `xorm:"created"`
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

// Validate 構造体を検証します
func (clip *Clip) Validate() error {
	return validator.ValidateStruct(clip)
}

// ClipMessage クリップメッセージ構造体
type ClipMessage struct {
	*Clip    `xorm:"extends"`
	*Message `xorm:"extends"`
}

// TableName Join処理用
func (*ClipMessage) TableName() string {
	return "clips"
}

// GetClipFolder 指定したIDのクリップフォルダを取得します
func GetClipFolder(id uuid.UUID) (*ClipFolder, error) {
	f := &ClipFolder{}
	ok, err := db.ID(id.String()).Get(f)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return f, nil
}

// GetClipFolders 指定したユーザーのクリップフォルダを全て取得します
func GetClipFolders(userID uuid.UUID) (res []*ClipFolder, err error) {
	err = db.Where("user_id = ?", userID.String()).Find(&res)
	return
}

// CreateClipFolder クリップフォルダを作成します
func CreateClipFolder(userID uuid.UUID, name string) (*ClipFolder, error) {
	f := &ClipFolder{
		ID:     CreateUUID(),
		UserID: userID.String(),
		Name:   name,
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	_, err := db.InsertOne(f)
	return f, err
}

// UpdateClipFolder クリップフォルダを更新します
func UpdateClipFolder(f *ClipFolder) (err error) {
	if err = f.Validate(); err != nil {
		return err
	}
	_, err = db.ID(f.ID).Update(f)
	return
}

// DeleteClipFolder クリップフォルダを削除します
func DeleteClipFolder(id uuid.UUID) (err error) {
	f := &ClipFolder{
		ID: id.String(),
	}
	_, err = db.Delete(f)
	return
}

func getClipMessageJoinedSession() *xorm.Session {
	return db.Join("INNER", "messages", "clips.message_id = messages.id AND messages.is_deleted = false")
}

// GetClipMessage 指定したIDのクリップを取得します
func GetClipMessage(id uuid.UUID) (*ClipMessage, error) {
	c := &ClipMessage{}
	ok, err := getClipMessageJoinedSession().Where("clips.id = ?", id.String()).Get(c)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}

// GetClipMessages 指定したフォルダのクリップを全て取得します
func GetClipMessages(folderID uuid.UUID) (res []*ClipMessage, err error) {
	err = getClipMessageJoinedSession().Where("clips.folder_id = ?", folderID.String()).Find(&res)
	return
}

// GetClipMessagesByUser 指定したユーザーのクリップを全て取得します
func GetClipMessagesByUser(userID uuid.UUID) (res []*ClipMessage, err error) {
	err = getClipMessageJoinedSession().Where("clips.user_id = ?", userID.String()).Find(&res)
	return
}

// CreateClip クリップを作成します
func CreateClip(messageID, folderID, userID uuid.UUID) (*Clip, error) {
	c := &Clip{
		ID:        CreateUUID(),
		UserID:    userID.String(),
		MessageID: messageID.String(),
		FolderID:  folderID.String(),
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	_, err := db.InsertOne(c)
	return c, err
}

// UpdateClip クリップを更新します
func UpdateClip(c *Clip) (err error) {
	if err = c.Validate(); err != nil {
		return err
	}
	_, err = db.ID(c.ID).Update(c)
	return
}

// DeleteClip クリップを削除します
func DeleteClip(id uuid.UUID) (err error) {
	c := &Clip{
		ID: id.String(),
	}
	_, err = db.Delete(c)
	return
}
