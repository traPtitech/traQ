package repository

// Repository データリポジトリ
type Repository interface {
	Sync() (bool, error)
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
}
