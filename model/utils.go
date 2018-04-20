package model

import (
	"errors"
	"fmt"
	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/rbac/role"
)

var (
	db *xorm.Engine

	// モデルを追加したら各自ここに追加しなければいけない
	// **順番注意**
	tables = []interface{}{
		&OAuth2Token{},
		&OAuth2Authorize{},
		&OAuth2Client{},
		&UserInvisibleChannel{},
		&RBACOverride{},
		&Webhook{},
		&Bot{},
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
	constraints = [][6]string{
		// Table, Key, ReferenceTable, ReferenceColumn, OnDelete, OnUpdate
		{"channels", "creator_id", "users", "id", "CASCADE", "CASCADE"},
		{"channels", "updater_id", "users", "id", "CASCADE", "CASCADE"},
		{"users_private_channels", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"users_private_channels", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"messages", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"messages", "updater_id", "users", "id", "CASCADE", "CASCADE"},
		{"messages", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"users_tags", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"users_tags", "tag_id", "tags", "id", "CASCADE", "CASCADE"},
		{"unreads", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"unreads", "message_id", "messages", "id", "CASCADE", "CASCADE"},
		{"devices", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"files", "creator_id", "users", "id", "CASCADE", "CASCADE"},
		{"stars", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"stars", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"users_subscribe_channels", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"clips", "folder_id", "clip_folders", "id", "CASCADE", "CASCADE"},
		{"pins", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"pins", "message_id", "messages", "id", "CASCADE", "CASCADE"},
		{"pins", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"messages_stamps", "message_id", "messages", "id", "CASCADE", "CASCADE"},
		{"messages_stamps", "stamp_id", "stamps", "id", "CASCADE", "CASCADE"},
		{"messages_stamps", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"stamps", "creator_id", "users", "id", "CASCADE", "CASCADE"},
		{"stamps", "file_id", "files", "id", "CASCADE", "CASCADE"},
		{"users_invisible_channels", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"users_invisible_channels", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"bots", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"bots", "creator_id", "users", "id", "CASCADE", "CASCADE"},
		{"bots", "updater_id", "users", "id", "CASCADE", "CASCADE"},
		{"webhooks", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"webhooks", "channel_id", "channels", "id", "CASCADE", "CASCADE"},
		{"rbac_overrides", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"oauth2_clients", "creator_id", "users", "id", "CASCADE", "CASCADE"},
		{"oauth2_authorizes", "client_id", "oauth2_clients", "id", "CASCADE", "CASCADE"},
		{"oauth2_authorizes", "user_id", "users", "id", "CASCADE", "CASCADE"},
		{"oauth2_tokens", "user_id", "users", "id", "CASCADE", "CASCADE"},
	}

	serverUser *User

	// ErrNotFoundOrForbidden 汎用エラー: 見つからないかスコープ外にある場合のエラー
	ErrNotFoundOrForbidden = errors.New("not found or forbidden")
	// ErrNotFound 汎用エラー: 見つからない場合のエラー
	ErrNotFound = errors.New("not found")
	// ErrInvalidParam 汎用エラー: データが不足・間違っている場合のエラー
	ErrInvalidParam = errors.New("invalid parameter")
)

func addForeignKeyConstraint(table, key, referenceTable, referenceColumn, onDelete, onUpdate string) error {
	switch onDelete {
	case "RESTRICT", "CASCADE", "SET NULL", "NO ACTION":
		break
	default:
		return errors.New("invalid reference option")
	}
	switch onUpdate {
	case "RESTRICT", "CASCADE", "SET NULL", "NO ACTION":
		break
	default:
		return errors.New("invalid reference option")
	}

	constName := fmt.Sprintf("%s_ibfk_%s__%s__%s", table, key, referenceTable, referenceColumn)

	c, err := db.SQL(`SELECT * FROM information_schema.table_constraints WHERE table_schema = ? AND constraint_type = 'FOREIGN KEY' AND constraint_name = ?`, config.DatabaseName, constName).Count()
	if err != nil {
		return err
	}
	if c > 0 {
		return nil
	}

	if _, err := db.Exec("ALTER TABLE ? ADD CONSTRAINT ? FOREIGN KEY ("+key+") REFERENCES ? (?) ON DELETE "+onDelete+" ON UPDATE "+onUpdate, table, constName, referenceTable, referenceColumn); err != nil {
		return err
	}

	return nil
}

// SetXORMEngine DBにxormのエンジンを設定する
func SetXORMEngine(engine *xorm.Engine) {
	db = engine
}

// SyncSchema : テーブルと構造体を同期させる
func SyncSchema() error {
	if err := db.Sync(tables...); err != nil {
		return fmt.Errorf("failed to sync Table schema: %v", err)
	}

	for _, c := range constraints {
		if err := addForeignKeyConstraint(c[0], c[1], c[2], c[3], c[4], c[5]); err != nil {
			return err
		}
	}
	// TODO: 初回起動時にgeneralチャンネルを作りたい

	serverUser = &User{Name: "traq", Email: "trap.titech@gmail.com", Role: role.Admin.ID()}
	if ok, err := serverUser.Exists(); err != nil {
		return err
	} else if !ok {
		serverUser.SetPassword("traq")
		serverUser.ID = CreateUUID()
		serverUser.Status = 1 // TODO: 状態確認
		if _, err := db.Insert(serverUser); err != nil {
			return err
		}
	}
	return nil
}

// DropTables : 全てのテーブルを削除する
func DropTables() error {
	//外部キー制約がかかってるので削除する順番に注意
	for _, v := range tables {
		if err := db.DropTables(v); err != nil {
			return err
		}
	}
	return nil
}

// CreateUUID UUIDを生成する
func CreateUUID() string {
	return uuid.NewV4().String()
}

// InitCache : 各種キャッシュを初期化する
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
		channelPathMap.Store(uuid.FromStringOrNil(v.ID), path)
	}

	// サムネイル未作成なファイルのサムネイル作成を試みる
	var files []*File
	if err := db.Where("is_deleted = false AND has_thumbnail = false").Find(&files); err != nil {
		return err
	}
	for _, f := range files {
		f.RegenerateThumbnail()
	}

	return nil
}
