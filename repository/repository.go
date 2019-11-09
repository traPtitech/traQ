package repository

import (
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/storage"
)

// Repository データリポジトリ
type Repository interface {
	// Sync DBなどとデータを同期します
	//
	// 返り値がtrueの場合、traqユーザーが作成されました。
	// DBによるエラーを返すことがあります。
	Sync() (bool, error)
	// GetFS ファイルストレージを取得します
	GetFS() storage.FileStorage
	UserRepository
	UserGroupRepository
	TagRepository
	ChannelRepository
	MessageRepository
	MessageReportRepository
	MessageStampRepository
	StampRepository
	ClipRepository
	StarRepository
	PinRepository
	DeviceRepository
	FileRepository
	WebhookRepository
	OAuth2Repository
	BotRepository
	UserRoleRepository
	message.ReplaceMapper
}
