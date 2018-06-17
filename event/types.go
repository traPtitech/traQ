package event

// Type イベントの種類
type Type string

const (
	// UserJoined ユーザーが加入した
	UserJoined Type = "USER_JOINED"
	// UserLeft ユーザーが脱退した
	UserLeft Type = "USER_LEFT"
	// UserUpdated ユーザーの情報が更新された
	UserUpdated Type = "USER_UPDATED"
	// UserTagsUpdated ユーザーのタグが更新された
	UserTagsUpdated Type = "USER_TAGS_UPDATED"
	// UserIconUpdated ユーザーのアイコンが更新された
	UserIconUpdated Type = "USER_ICON_UPDATED"
	// UserOnline ユーザーがオンラインになった
	UserOnline Type = "USER_ONLINE"
	// UserOffline ユーザーがオフラインになった
	UserOffline Type = "USER_OFFLINE"

	// ChannelCreated チャンネルが新規作成された
	ChannelCreated Type = "CHANNEL_CREATED"
	// ChannelDeleted チャンネルが削除された
	ChannelDeleted Type = "CHANNEL_DELETED"
	// ChannelUpdated チャンネルの名前またはトピックが変更された
	ChannelUpdated Type = "CHANNEL_UPDATED"
	// ChannelStared チャンネルをスターした
	ChannelStared Type = "CHANNEL_STARED"
	// ChannelUnstared チャンネルのスターを解除した
	ChannelUnstared Type = "CHANNEL_UNSTARED"
	// ChannelVisibilityChanged チャンネルの可視状態が変更された
	ChannelVisibilityChanged Type = "CHANNEL_VISIBILITY_CHANGED"

	// MessageCreated メッセージが投稿された
	MessageCreated Type = "MESSAGE_CREATED"
	// MessageUpdated メッセージが更新された
	MessageUpdated Type = "MESSAGE_UPDATED"
	// MessageDeleted メッセージが削除された
	MessageDeleted Type = "MESSAGE_DELETED"
	// MessageRead メッセージを読んだ
	MessageRead Type = "MESSAGE_READ"
	// MessageStamped メッセージにスタンプが押された
	MessageStamped Type = "MESSAGE_STAMPED"
	// MessageUnstamped メッセージからスタンプが外された
	MessageUnstamped Type = "MESSAGE_UNSTAMPED"
	// MessagePinned メッセージがピン留めされた
	MessagePinned Type = "MESSAGE_PINNED"
	// MessageUnpinned ピン留めされたメッセージのピンが外された
	MessageUnpinned Type = "MESSAGE_UNPINNED"

	// ClipCreated メッセージをクリップした
	ClipCreated Type = "CLIP_CREATED"
	// ClipDeleted メッセージをアンクリップした
	ClipDeleted Type = "CLIP_DELETED"
	// ClipMoved クリップのフォルダが変更された
	ClipMoved Type = "CLIP_MOVED"
	// ClipFolderCreated クリップフォルダが作成された
	ClipFolderCreated Type = "CLIP_FOLDER_CREATED"
	// ClipFolderUpdated クリップフォルダが更新された
	ClipFolderUpdated Type = "CLIP_FOLDER_UPDATED"
	// ClipFolderDeleted クリップフォルダが削除された
	ClipFolderDeleted Type = "CLIP_FOLDER_DELETED"

	// StampCreated スタンプが新しく追加された
	StampCreated Type = "STAMP_CREATED"
	// StampModified スタンプが修正された
	StampModified Type = "STAMP_MODIFIED"
	// StampDeleted スタンプが削除された
	StampDeleted Type = "STAMP_DELETED"

	// TraqUpdated traQが更新された
	TraqUpdated Type = "TRAQ_UPDATED"
)
