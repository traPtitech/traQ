package model

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
)

const (
	errMySQLDuplicatedRecord uint16 = 1062
)

var (
	db *gorm.DB

	// モデルを追加したら各自ここに追加しなければいけない
	// **順番注意**
	tables = []interface{}{
		&Mute{},
		&MessageReport{},
		&OAuth2Token{},
		&OAuth2Authorize{},
		&OAuth2Client{},
		&RBACOverride{},
		&BotOutgoingPostLog{},
		&BotInstalledChannel{},
		&GeneralBot{},
		&WebhookBot{},
		&MessageStamp{},
		&Stamp{},
		&Clip{},
		&ClipFolder{},
		&UsersTag{},
		&Unread{},
		&Star{},
		&Device{},
		&Pin{},
		&File{},
		&UsersPrivateChannel{},
		&UserSubscribeChannel{},
		&Tag{},
		&Message{},
		&Channel{},
		&User{},
	}

	// 外部キー制約
	constraints = [][5]string{
		// Table, Key, Reference, OnDelete, OnUpdate
		{"channels", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"channels", "updater_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_private_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_private_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"messages", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"messages", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"users_tags", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_tags", "tag_id", "tags(id)", "CASCADE", "CASCADE"},
		{"unreads", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"unreads", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"devices", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"files", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"stars", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"stars", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"clips", "folder_id", "clip_folders(id)", "CASCADE", "CASCADE"},
		{"clips", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"clips", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"clip_folders", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"pins", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"pins", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "stamp_id", "stamps(id)", "CASCADE", "CASCADE"},
		{"messages_stamps", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"stamps", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"stamps", "file_id", "files(id)", "CASCADE", "CASCADE"},
		{"bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
		{"webhook_bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
		{"rbac_overrides", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"oauth2_clients", "creator_id", "users(id)", "CASCADE", "CASCADE"},
		{"oauth2_authorizes", "client_id", "oauth2_clients(id)", "CASCADE", "CASCADE"},
		{"oauth2_authorizes", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"oauth2_tokens", "user_id", "users(id)", "CASCADE", "CASCADE"},
	}

	serverUser = &User{}

	// ErrNotFoundOrForbidden 汎用エラー: 見つからないかスコープ外にある場合のエラー
	ErrNotFoundOrForbidden = errors.New("not found or forbidden")
	// ErrNotFound 汎用エラー: 見つからない場合のエラー
	ErrNotFound = errors.New("not found")
)

// SetGORMEngine DBにgormのエンジンを設定する
func SetGORMEngine(engine *gorm.DB) {
	db = engine
}

// Sync テーブルと構造体を同期させる
func Sync() (bool, error) {
	// スキーマ同期
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(tables...).Error; err != nil {
		return false, fmt.Errorf("failed to sync Table schema: %v", err)
	}

	// 外部キー制約同期
	for _, c := range constraints {
		if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
			return false, err
		}
	}

	// サーバーユーザーの確認
	if err := db.Where(User{Name: "traq"}).Take(serverUser).Error; err != nil {
		if !gorm.IsRecordNotFoundError(err) {
			return false, err
		}

		// サーバーユーザーが存在しない場合は作成
		salt := generateSalt()
		serverUser = &User{
			ID:       CreateUUID(),
			Name:     "traq",
			Password: hex.EncodeToString(hashPassword("traq", salt)),
			Salt:     hex.EncodeToString(salt),
			Role:     role.Admin.ID(),
		}
		if err := db.Create(serverUser).Error; err != nil {
			return false, err
		}
		fileID, err := GenerateIconFile(uuid.NewV4().String())
		if err != nil {
			return false, err
		}
		if err := ChangeUserIcon(serverUser.GetUID(), fileID); err != nil {
			return false, err
		}

		return true, nil
	}
	return false, nil
}

// DropTables 全てのテーブルを削除する
func DropTables() error {
	//外部キー制約がかかってるので削除する順番に注意
	for _, v := range tables {
		if err := db.DropTable(v).Error; err != nil {
			return err
		}
	}
	return nil
}

// CreateUUID UUIDを生成する
func CreateUUID() string {
	return uuid.NewV4().String()
}

// InitCache 各種キャッシュを初期化する
func InitCache() error {
	channels, err := GetAllChannels()
	if err != nil {
		return err
	}
	for _, v := range channels {
		path, err := v.Path()
		if err != nil {
			return err
		}
		channelPathMap.Store(v.ID, path)
	}

	// サムネイル未作成なファイルのサムネイル作成を試みる
	var files []*File
	if err := db.Where("has_thumbnail = false").Find(&files).Error; err != nil {
		return err
	}
	for _, f := range files {
		f.RegenerateThumbnail()
	}

	return nil
}

func convertStringSliceToUUIDSlice(arr []string) (result []uuid.UUID) {
	result = make([]uuid.UUID, len(arr))
	for i, v := range arr {
		result[i] = uuid.Must(uuid.FromString(v))
	}
	return
}

func isMySQLDuplicatedRecordErr(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLDuplicatedRecord
}

func transact(txFunc func(tx *gorm.DB) error) (err error) {
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()
	err = txFunc(tx)
	return err
}

// ServerUser サーバーユーザーを返します
func ServerUser() *User {
	return serverUser
}
