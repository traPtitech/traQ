package model

import (
	"fmt"

	"errors"

	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

var (
	db *xorm.Engine

	// モデルを追加したら各自ここに追加しなければいけない
	// **順番注意**
	tables = []interface{}{
		&UserInvisibleChannel{},
		&Webhook{},
		&Bot{},
		&MessageStamp{},
		&Stamp{},
		&Clip{},
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

	serverUser *User

	// ErrNotFoundOrForbidden 汎用エラー: 見つからないかスコープ外にある場合のエラー
	ErrNotFoundOrForbidden = errors.New("not found or forbidden")
	// ErrNotFound 汎用エラー: 見つからない場合のエラー
	ErrNotFound = errors.New("not found")
	// ErrInvalidParam 汎用エラー: データが不足・間違っている場合のエラー
	ErrInvalidParam = errors.New("invalid parameter")
)

// SetXORMEngine DBにxormのエンジンを設定する
func SetXORMEngine(engine *xorm.Engine) {
	db = engine
}

// SyncSchema : テーブルと構造体を同期させる
func SyncSchema() error {
	if err := db.Sync(tables...); err != nil {
		return fmt.Errorf("failed to sync Table schema: %v", err)
	}
	if _, err := db.Exec("ALTER TABLE `channels` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `channels` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_private_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_private_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_tags` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_tags` ADD FOREIGN KEY (`tag_id`) REFERENCES `tags`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `unreads` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `unreads` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `devices` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `files` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `stars` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `stars` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_subscribe_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_subscribe_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `clips` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `clips` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `pins` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `pins` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `pins` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `clips` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`stamp_id`) REFERENCES `stamps`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `stamps` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `stamps` ADD FOREIGN KEY (`file_id`) REFERENCES `files`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_invisible_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `users_invisible_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `bots` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `bots` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `bots` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `webhooks` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE `webhooks` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;"); err != nil {
		return err
	}

	traq := &User{
		Name:  "traq",
		Email: "trap.titech@gmail.com",
	}
	ok, err := traq.Exists()
	if err != nil {
		return err
	}
	if !ok {
		traq.SetPassword("traq")
		traq.ID = CreateUUID()
		traq.Icon = ""
		if _, err := db.Insert(traq); err != nil {
			return err
		}
	}
	serverUser = traq

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
