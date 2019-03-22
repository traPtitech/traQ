package model

var (
	// Tables データベースのテーブルモデル
	// モデルを追加したら各自ここに追加しなければいけない
	// **順番注意**
	Tables = []interface{}{
		&OAuth2Client{},
		&OAuth2Authorize{},
		&OAuth2Token{},
		&Mute{},
		&MessageReport{},
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
		&FileACLEntry{},
		&File{},
		&UsersPrivateChannel{},
		&UserSubscribeChannel{},
		&Tag{},
		&ArchivedMessage{},
		&Message{},
		&Channel{},
		&UserGroupMember{},
		&UserGroup{},
		&User{},
	}

	// Constraints 外部キー制約
	Constraints = [][5]string{
		// Table, Key, Reference, OnDelete, OnUpdate
		{"users_private_channels", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_private_channels", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"messages", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"messages", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
		{"users_tags", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"users_tags", "tag_id", "tags(id)", "CASCADE", "CASCADE"},
		{"unreads", "user_id", "users(id)", "CASCADE", "CASCADE"},
		{"unreads", "message_id", "messages(id)", "CASCADE", "CASCADE"},
		{"devices", "user_id", "users(id)", "CASCADE", "CASCADE"},
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
		{"webhook_bots", "bot_user_id", "users(id)", "CASCADE", "CASCADE"},
	}
)
