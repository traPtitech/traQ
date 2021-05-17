package repository

// Repository データリポジトリ
type Repository interface {
	// Sync DBなどとデータを同期します
	//
	// スキーマが初期化された場合、trueを返します。
	// DBによるエラーを返すことがあります。
	Sync() (bool, error)
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
}
