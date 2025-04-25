package repository

// Repository データリポジトリ
type Repository interface {
	UserRepository
	UserGroupRepository
	UserSettingsRepository
	UserRoleRepository
	TagRepository
	ChannelRepository
	MessageRepository
	MessageReportRepository
	StampRepository
	StampPaletteRepository
	StarRepository
	PinRepository
	DeviceRepository
	FileRepository
	WebhookRepository
	OAuth2Repository
	BotRepository
	ClipRepository
	OgpCacheRepository
	SoundboardRepository
}
