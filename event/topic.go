package event

const (
	// UserCreated ユーザーが追加された
	// 	Fields:
	//      user_id: uuid.UUID
	// 		user: *model.User
	UserCreated = "user.created"
	// UserUpdated ユーザーが更新された
	// 	Fields:
	//      user_id: uuid.UUID
	UserUpdated = "user.updated"
	// UserIconUpdated ユーザーのアイコンが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		file_id: uuid.UUID
	UserIconUpdated = "user.icon_updated"
	// UserOnline ユーザーがオンラインになった
	// 	Fields:
	//      user_id: uuid.UUID
	UserOnline = "user.online"
	// UserOffline ユーザーがオフラインになった
	// 	Fields:
	//      user_id: uuid.UUID
	// 		datetime: time.Time
	UserOffline = "user.offline"

	// UserTagAdded ユーザーにタグが追加された
	// 	Fields:
	//      user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagAdded = "user_tag.added"
	// UserTagUpdated ユーザーのタグが更新された
	// 	Fields:
	//      user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagUpdated = "user_tag.updated"
	// UserTagRemoved ユーザーからタグが削除された
	// 	Fields:
	//      user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagRemoved = "user_tag.deleted"

	// MessageCreated メッセージが作成された
	// 	Fields:
	// 		message_id: uuid.UUID
	//  	message: *model.Message
	//  	embedded: []*message.EmbeddedInfo
	//      plain: string
	MessageCreated = "message.created"
	// MessageUpdated メッセージが更新された
	// 	Fields:
	// 		message_id: uuid.UUID
	//  	message: *model.Message
	//  	old_message: *model.Message
	MessageUpdated = "message.updated"
	// MessageDeleted メッセージが削除された
	// 	Fields:
	// 		message_id: uuid.UUID
	//  	message: *model.Message
	MessageDeleted = "message.deleted"
	// MessageStamped メッセージにスタンプが押された
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		user_id: uuid.UUID
	// 		stamp_id: uuid.UUID
	// 		count: int
	// 		created_at: time.Time
	MessageStamped = "message.stamped"
	// MessageUnstamped メッセージからスタンプが消された
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		user_id: uuid.UUID
	// 		stamp_id: uuid.UUID
	MessageUnstamped = "message.unstamped"
	// MessagePinned メッセージがピンされた
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		pin_id: uuid.UUID
	MessagePinned = "message.pinned"
	// MessageUnpinned メッセージがピンから外れた
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		pin_id: uuid.UUID
	MessageUnpinned = "message.unpinned"

	// ChannelCreated チャンネルが作成された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		private: bool
	ChannelCreated = "channel.created"
	// ChannelUpdated チャンネルが更新された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		private: bool
	ChannelUpdated = "channel.updated"
	// ChannelDeleted チャンネルが削除された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		private: bool
	ChannelDeleted = "channel.deleted"
	// ChannelRead チャンネルのメッセージが既読された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelRead = "channel.read"
	// ChannelStared チャンネルがスターされた
	// 	Fields:
	//		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelStared = "channel.stared"
	// ChannelUnstared チャンネルのスターが解除された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelUnstared = "channel.unstared"
	// ChannelMuted チャンネルがミュートされた
	// 	Fields:
	//		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelMuted = "channel.muted"
	// ChannelUnmuted チャンネルのミュートが解除された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelUnmuted = "channel.unmuted"

	// StampCreated スタンプが作成された
	// 	Fields:
	// 		stamp_id: uuid.UUID
	// 		stamp: *model.Stamp
	StampCreated = "stamp.created"
	// StampUpdated スタンプが更新された
	// 	Fields:
	// 		stamp_id: uuid.UUID
	StampUpdated = "stamp.updated"
	// StampDeleted スタンプが削除された
	// 	Fields:
	// 		stamp_id: uuid.UUID
	StampDeleted = "stamp.deleted"

	// ClipCreated クリップが作成された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		clip_id: uuid.UUID
	ClipCreated = "clip.created"
	// ClipMoved クリップが移動された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		clip_id: uuid.UUID
	ClipMoved = "clip.moved"
	// ClipDeleted クリップが削除された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		clip_id: uuid.UUID
	ClipDeleted = "clip.deleted"
	// ClipFolderCreated クリップフォルダが作成された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		folder_id: uuid.UUID
	ClipFolderCreated = "clip_folder.created"
	// ClipFolderUpdated クリップフォルダが更新された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		folder_id: uuid.UUID
	ClipFolderUpdated = "clip_folder.updated"
	// ClipFolderDeleted クリップフォルダが削除された
	// 	Fields:
	//		user_id: uuid.UUID
	// 		folder_id: uuid.UUID
	ClipFolderDeleted = "clip_folder.deleted"
)
