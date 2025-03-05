package event

const (
	// UserCreated ユーザーが追加された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		user: *model.User
	UserCreated = "user.created"
	// UserUpdated ユーザーが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	UserUpdated = "user.updated"
	// UserIconUpdated ユーザーのアイコンが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		file_id: uuid.UUID
	UserIconUpdated = "user.icon_updated"
	// UserOnline ユーザーがオンラインになった
	// 	Fields:
	// 		user_id: uuid.UUID
	UserOnline = "user.online"
	// UserOffline ユーザーがオフラインになった
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		datetime: time.Time
	UserOffline = "user.offline"
	// UserActivated ユーザーの凍結が解除された
	// 	Fields:
	// 		user: *model.User
	UserActivated = "user.activated"
	// UserViewStateChanged ユーザーの閲覧状態が変化した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		view_states: map[string]viewer.StateWithChannel
	UserViewStateChanged = "user.viewstate.changed"

	// UserTagAdded ユーザーにタグが追加された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagAdded = "user_tag.added"
	// UserTagUpdated ユーザーのタグが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagUpdated = "user_tag.updated"
	// UserTagRemoved ユーザーからタグが削除された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		tag_id: uuid.UUID
	UserTagRemoved = "user_tag.deleted"

	// UserGroupCreated ユーザーグループが作成された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		group: *model.UserGroup
	UserGroupCreated = "user_group.created"
	// UserGroupUpdated ユーザーグループが更新された
	// 	Fields:
	// 		group_id: uuid.UUID
	UserGroupUpdated = "user_group.updated"
	// UserGroupDeleted ユーザーグループが削除された
	// 	Fields:
	// 		group_id: uuid.UUID
	UserGroupDeleted = "user_group.deleted"
	// UserGroupMemberAdded ユーザーがグループに追加された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		user_id: uuid.UUID
	UserGroupMemberAdded = "user_group.member.added"
	// UserGroupMemberUpdated ユーザーグループメンバーが更新された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		user_id: uuid.UUID
	UserGroupMemberUpdated = "user_group.member.updated"
	// UserGroupMemberRemoved ユーザーがグループから削除された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		user_id: uuid.UUID
	UserGroupMemberRemoved = "user_group.member.removed"
	// UserGroupAdminAdded ユーザーがグループの管理者に追加された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		user_id: uuid.UUID
	UserGroupAdminAdded = "user_group.admin.added"
	// UserGroupAdminRemoved ユーザーがグループの管理者から削除された
	// 	Fields:
	// 		group_id: uuid.UUID
	// 		user_id: uuid.UUID
	UserGroupAdminRemoved = "user_group.admin.removed"

	// MessageCreated メッセージが作成された
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		message: *model.Message
	// 		parse_result: *message.ParseResult
	MessageCreated = "message.created"
	// MessageUpdated メッセージが更新された
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		message: *model.Message
	// 		old_message: *model.Message
	MessageUpdated = "message.updated"
	// MessageDeleted メッセージが削除された
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		message: *model.Message
	// 		deleted_unreads: []*model.Unread
	MessageDeleted = "message.deleted"
	// MessageUnread メッセージが未読になった
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		user_id: uuid.UUID
	// 		noticeable: bool
	MessageUnread = "message.unread"
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
	// 		channel_id: uuid.UUID
	MessagePinned = "message.pinned"
	// MessageUnpinned メッセージがピンから外れた
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		channel_id: uuid.UUID
	MessageUnpinned = "message.unpinned"
	// MessageCited メッセージが引用された
	// 	Fields:
	// 		message_id: uuid.UUID	引用したメッセージのID
	// 		message: *model.Message
	// 		cited_ids: []uuid.UUID	引用されたメッセージのIDの配列
	MessageCited = "message.cited"

	// ChannelCreated チャンネルが作成された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		channel: *model.Channel
	// 		private: bool
	ChannelCreated = "channel.created"
	// ChannelUpdated チャンネルが更新された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		private: bool
	ChannelUpdated = "channel.updated"
	// ChannelTopicUpdated チャンネルトピックが更新された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		topic: string
	// 		updater_id: uuid.UUID
	ChannelTopicUpdated = "channel.topic.updated"
	// ChannelDeleted チャンネルが削除された
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		private: bool
	ChannelDeleted = "channel.deleted"
	// ChannelRead チャンネルのメッセージが既読された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	// 		read_messages_num: int
	ChannelRead = "channel.read"
	// ChannelStared チャンネルがスターされた
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelStared = "channel.stared"
	// ChannelUnstared チャンネルのスターが解除された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	ChannelUnstared = "channel.unstared"
	// ChannelViewersChanged チャンネルの閲覧者が変化した
	// 	Fields:
	// 		channel_id: uuid.UUID
	// 		viewers: map[uuid.UUID]viewer.StateWithTime
	ChannelViewersChanged = "channel.viewers_changed"
	// ChannelSubscribersChanged チャンネルの購読者が変化した
	// 	Fields:
	// 		channel_id: uuid.UUID
	//    subscriber_ids: []uuid.UUID
	ChannelSubscribersChanged = "channel.subscribers_changed"

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

	// StampPaletteCreated スタンプパレットが作成された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		stamp_palette_id: uuid.UUID
	// 		stamp_palette: *model.StampPalette
	StampPaletteCreated = "stamp_palette.created"
	// StampPaletteUpdated スタンプパレットが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		stamp_palette_id: uuid.UUID
	StampPaletteUpdated = "stamp_palette.updated"
	// StampPaletteDeleted スタンプパレットが削除された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		stamp_palette_id: uuid.UUID
	StampPaletteDeleted = "stamp_palette.deleted"

	// WebhookCreated Webhookが作成された
	// 	Fields:
	// 		webhook_id: uuid.UUID
	// 		webhook: Webhook
	WebhookCreated = "webhook.created"
	// WebhookUpdated Webhookが更新された
	// 	Fields:
	// 		webhook_id: uuid.UUID
	WebhookUpdated = "webhook.updated"
	// WebhookDeleted Webhookが削除された
	// 	Fields:
	// 		webhook_id: uuid.UUID
	WebhookDeleted = "webhook.deleted"

	// BotCreated Botが作成された
	// 	Fields:
	// 		bot_id: uuid.UUID
	// 		bot: *model.Bot
	BotCreated = "bot.created"
	// BotUpdated Botが更新された
	// 	Fields:
	// 		bot_id: uuid.UUID
	BotUpdated = "bot.updated"
	// BotDeleted Botが削除された
	// 	Fields:
	// 		bot_id: uuid.UUID
	BotDeleted = "bot.deleted"
	// BotStateChanged Botの状態が変化した
	// 	Fields:
	// 		bot_id: uuid.UUID
	// 		state: model.BotState
	BotStateChanged = "bot.state_changed"
	// BotPingRequest BotのPingがリクエストされた
	// 	Fields:
	// 		bot_id: uuid.UUID
	// 		bot: *model.Bot
	BotPingRequest = "bot.ping"
	// BotJoined Botがチャンネルに参加した
	// 	Fields:
	// 		bot_id: uuid.UUID
	// 		channel_id: uuid.UUID
	BotJoined = "bot.joined"
	// BotLeft Botがチャンネルから退出した
	// 	Fields:
	// 		bot_id: uuid.UUID
	// 		channel_id: uuid.UUID
	BotLeft = "bot.left"

	// UserWebRTCv3StateChanged ユーザーのWebRTCの状態が変化した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		channel_id: uuid.UUID
	// 		sessions: map[string]string
	UserWebRTCv3StateChanged = "user.webrtc_v3.state_changed"

	// QallRoomStateChanged Qallのルーム状態が変更された
	// 	Fields:
	// 		room_id: uuid.UUID
	// 		state: *qall.RoomWithParticipants
	QallRoomStateChanged = "qall.roomstate.changed"

	// QallSoundboardItemCreated サウンドボードアイテムが作成された
	// 	Fields:
	// 		sound_id: uuid.UUID
	// 		name: string
	// 		creator_id: uuid.UUID
	QallSoundboardItemCreated = "qall.soundboard.item.created"

	// QallSoundboardItemDeleted サウンドボードアイテムが削除された
	// 	Fields:
	// 		sound_id: uuid.UUID
	QallSoundboardItemDeleted = "qall.soundboard.item.deleted"

	// WSConnected ユーザーがWSストリームに接続した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		req: *http.Request
	WSConnected = "ws.connected"
	// WSDisconnected ユーザーがWSストリームから切断した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		req: *http.Request
	WSDisconnected = "ws.disconnected"

	// BotWSConnected BOTユーザーがWSストリームに接続した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		req: *http.Request
	BotWSConnected = "bot.ws.connected"
	// BotWSDisconnected BOTユーザーがWSストリームから切断した
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		req: *http.Request
	BotWSDisconnected = "bot.ws.disconnected"

	// ClipFolderCreated クリップフォルダーが作成された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		clip_folder_id: uuid.UUID
	// 		clip_folder: *model.ClipFolder
	ClipFolderCreated = "clip_folder.created"
	// ClipFolderUpdated クリップフォルダーが更新された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		clip_folder_id: uuid.UUID
	// 		old_clip_folder: *model.ClipFolder
	// 		clip_folder: *model.ClipFolder
	ClipFolderUpdated = "clip_folder.updated"
	// ClipFolderDeleted クリップフォルダーが削除された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		clip_folder_id: uuid.UUID
	// 		clip_folder: *model.ClipFolder
	ClipFolderDeleted = "clip_folder.deleted"
	// ClipFolderMessageDeleted クリップフォルダーのメッセージが除外された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		clip_folder_id: uuid.UUID
	// 		clip_folder_message_id: uuid.UUID
	// 		clip_folder_message: *model.ClipFolderMessage
	ClipFolderMessageDeleted = "clip_folder_message.deleted"
	// ClipFolderMessageAdded クリップフォルダーにメッセージが追加された
	// 	Fields:
	// 		user_id: uuid.UUID
	// 		clip_folder_id: uuid.UUID
	// 		clip_folder_message_id: uuid.UUID
	// 		clip_folder_message: *model.ClipFolderMessage
	ClipFolderMessageAdded = "clip_folder_message.added"

	// MessageStampsUpdated メッセージに押されているスタンプが変化した。このイベントはスロットリングされています
	// 	Fields:
	// 		message_id: uuid.UUID
	// 		message: message.Message
	MessageStampsUpdated = "message.stamps.updated"
)
