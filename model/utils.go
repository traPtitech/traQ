package model

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
	"gopkg.in/go-playground/validator.v9"
)

var (
	db *xorm.Engine

	// モデルを追加したら各自ここに追加しなければいけない
	// **順番注意**
	tables = []interface{}{
		&UserInvisibleChannel{},
		&RBACOverride{},
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

	// 外部キー制約
	constraints = []string{
		"ALTER TABLE `channels` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `channels` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_private_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_private_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_tags` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_tags` ADD FOREIGN KEY (`tag_id`) REFERENCES `tags`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `unreads` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `unreads` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `devices` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `files` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `stars` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `stars` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_subscribe_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_subscribe_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `clips` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `clips` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `pins` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `pins` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `pins` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `clips` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`stamp_id`) REFERENCES `stamps`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `messages_stamps` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `stamps` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `stamps` ADD FOREIGN KEY (`file_id`) REFERENCES `files`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_invisible_channels` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `users_invisible_channels` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `bots` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `bots` ADD FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `bots` ADD FOREIGN KEY (`updater_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `webhooks` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `webhooks` ADD FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
		"ALTER TABLE `rbac_overrides` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE",
	}

	serverUser *User

	// ErrNotFoundOrForbidden : 汎用エラー 見つからないかスコープ外にある場合のエラー
	ErrNotFoundOrForbidden = errors.New("not found or forbidden")
	// ErrNotFound : 汎用エラー 見つからない場合のエラー
	ErrNotFound = errors.New("not found")

	validate *validator.Validate
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

	for _, sql := range constraints {
		if _, err := db.Exec(sql); err != nil {
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

func validateStruct(i interface{}) error {
	if validate == nil {
		validate = validator.New()

		name := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
		validate.RegisterValidation("name", func(fl validator.FieldLevel) bool {
			return name.MatchString(fl.Field().String())
		})
	}
	return validate.Struct(i)
}
