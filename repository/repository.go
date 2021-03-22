package repository

// Repository データリポジトリ
type Repository interface {
	// Sync DBなどとデータを同期します
	//
	// 返り値がtrueの場合、traqユーザーが作成されました。
	// DBによるエラーを返すことがあります。
	Sync() (bool, error)
	UserRepository
	UserGroupRepository
	UserSettingsRepository
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
