package repository

import "github.com/traPtitech/traQ/utils/storage"

// Repository データリポジトリ
type Repository interface {
	Sync() (bool, error)
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
	MuteRepository
	StarRepository
	PinRepository
	DeviceRepository
	FileRepository
	WebhookRepository
	OAuth2Repository
}
