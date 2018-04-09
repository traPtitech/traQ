package model

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/rbac/role"
	"net/url"
	"strings"
	"time"
)

// BotStoreImpl Botデータ用ストアの実装
type BotStoreImpl struct{}

// SavePostLog Botのポストログをdbに保存
func (s *BotStoreImpl) SavePostLog(reqID, botUserID uuid.UUID, status int, request, response, error string) error {
	l := &BotOutgoingPostLog{
		RequestID:  reqID.String(),
		BotUserID:  botUserID.String(),
		StatusCode: status,
		Request:    request,
		Response:   response,
		Error:      error,
		Timestamp:  time.Now().UnixNano(),
	}

	if _, err := db.InsertOne(l); err != nil {
		return err
	}
	return nil
}

// SavePlugin Pluginをdbに保存
func (s *BotStoreImpl) SavePlugin(bp *bot.Plugin) error {
	u := &User{
		ID:          bp.BotUserID.String(),
		Name:        "Plugin#" + base64.RawStdEncoding.EncodeToString(bp.BotUserID.Bytes()),
		DisplayName: bp.DisplayName,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        bp.IconFileID.String(),
		Status:      0, //TODO
		Bot:         true,
		BotType:     bot.TypePlugin,
		Role:        role.Bot.ID(), //FIXME
	}
	p := &Plugin{
		ID:                bp.ID.String(),
		BotUserID:         bp.BotUserID.String(),
		Description:       bp.Description,
		Command:           bp.Command,
		Usage:             bp.Usage,
		VerificationToken: bp.VerificationToken,
		AccessTokenID:     bp.AccessTokenID.String(),
		PostURL:           bp.PostURL.String(),
		Activated:         bp.Activated,
		IsValid:           bp.IsValid,
		CreatorID:         bp.CreatorID.String(),
		CreatedAt:         bp.CreatedAt,
		UpdatedAt:         bp.UpdatedAt,
	}

	if _, err := db.UseBool().Insert(u, p); err != nil {
		return err
	}

	return nil
}

// UpdatePlugin Pluginをdbに保存
func (s *BotStoreImpl) UpdatePlugin(b *bot.Plugin) error {
	if _, err := db.ID(b.BotUserID.String()).Update(&User{
		DisplayName: b.DisplayName,
		Icon:        b.IconFileID.String(),
	}); err != nil {
		return err
	}
	if _, err := db.ID(b.ID.String()).UseBool("is_valid", "activated").Update(&Plugin{
		Description:       b.Description,
		Usage:             b.Usage,
		VerificationToken: b.VerificationToken,
		AccessTokenID:     b.AccessTokenID.String(),
		PostURL:           b.PostURL.String(),
		Activated:         b.Activated,
		IsValid:           b.IsValid,
		CreatorID:         b.CreatorID.String(),
		UpdatedAt:         b.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

// GetAllPlugins Pluginを全て取得
func (s *BotStoreImpl) GetAllPlugins() (arr []bot.Plugin, err error) {
	var ps []*PluginUser
	if err := db.Join("INNER", "plugins", "plugins.bot_user_id = users.id").Find(&ps); err != nil {
		return nil, err
	}
	for _, v := range ps {
		postURL, _ := url.Parse(v.Plugin.PostURL)
		arr = append(arr, bot.Plugin{
			ID:                uuid.Must(uuid.FromString(v.Plugin.ID)),
			BotUserID:         uuid.Must(uuid.FromString(v.Plugin.BotUserID)),
			DisplayName:       v.User.DisplayName,
			Description:       v.Plugin.Description,
			Command:           v.Command,
			Usage:             v.Usage,
			IconFileID:        uuid.Must(uuid.FromString(v.User.Icon)),
			VerificationToken: v.Plugin.VerificationToken,
			AccessTokenID:     uuid.Must(uuid.FromString(v.Plugin.AccessTokenID)),
			PostURL:           *postURL,
			Activated:         v.Plugin.Activated,
			IsValid:           v.Plugin.IsValid,
			CreatorID:         uuid.Must(uuid.FromString(v.Plugin.CreatorID)),
			CreatedAt:         v.Plugin.CreatedAt,
			UpdatedAt:         v.Plugin.UpdatedAt,
		})
	}
	return
}

// SaveGeneralBot GeneralBotをdbに保存
func (s *BotStoreImpl) SaveGeneralBot(b *bot.GeneralBot) error {
	u := &User{
		ID:          b.BotUserID.String(),
		Name:        b.Name,
		DisplayName: b.DisplayName,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        b.IconFileID.String(),
		Status:      0, //TODO
		Bot:         true,
		BotType:     bot.TypeGeneral,
		Role:        role.Bot.ID(), //FIXME
	}
	gb := &GeneralBot{
		ID:                b.ID.String(),
		BotUserID:         b.BotUserID.String(),
		Description:       b.Description,
		VerificationToken: b.VerificationToken,
		AccessTokenID:     b.AccessTokenID.String(),
		PostURL:           b.PostURL.String(),
		SubscribeEvents:   strings.Join(b.SubscribeEvents, " "),
		Activated:         b.Activated,
		IsValid:           b.IsValid,
		OwnerID:           b.OwnerID.String(),
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
	}

	if _, err := db.UseBool().Insert(u, gb); err != nil {
		return err
	}

	return nil
}

// UpdateGeneralBot GeneralBotをdbに保存
func (s *BotStoreImpl) UpdateGeneralBot(b *bot.GeneralBot) error {
	if _, err := db.ID(b.BotUserID.String()).Update(&User{
		DisplayName: b.DisplayName,
		Icon:        b.IconFileID.String(),
	}); err != nil {
		return err
	}
	if _, err := db.ID(b.ID.String()).UseBool("is_valid", "activated").Update(&GeneralBot{
		Description:       b.Description,
		VerificationToken: b.VerificationToken,
		AccessTokenID:     b.AccessTokenID.String(),
		PostURL:           b.PostURL.String(),
		SubscribeEvents:   strings.Join(b.SubscribeEvents, " "),
		Activated:         b.Activated,
		IsValid:           b.IsValid,
		OwnerID:           b.OwnerID.String(),
		UpdatedAt:         b.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

// GetAllBotsInstalledChannels BotInstalledChannelを全て取得
func (s *BotStoreImpl) GetAllBotsInstalledChannels() (arr []bot.InstalledChannel, err error) {
	var bics []BotInstalledChannel
	if err = db.Find(&bics); err != nil {
		return nil, err
	}
	for _, v := range bics {
		arr = append(arr, bot.InstalledChannel{
			BotID:       uuid.Must(uuid.FromString(v.BotID)),
			ChannelID:   uuid.Must(uuid.FromString(v.ChannelID)),
			InstalledBy: uuid.Must(uuid.FromString(v.InstalledBy)),
		})
	}
	return
}

// InstallBot GeneralBotをチャンネルにインストール(db)
func (s *BotStoreImpl) InstallBot(botID, channelID, userID uuid.UUID) error {
	bic := &BotInstalledChannel{
		BotID:       botID.String(),
		ChannelID:   channelID.String(),
		InstalledBy: userID.String(),
	}

	if _, err := db.InsertOne(bic); err != nil {
		return err
	}
	return nil
}

// UninstallBot GeneralBotをチャンネルからアンインストール(db)
func (s *BotStoreImpl) UninstallBot(botID, channelID uuid.UUID) error {
	bic := &BotInstalledChannel{
		BotID:     botID.String(),
		ChannelID: channelID.String(),
	}

	if _, err := db.Delete(bic); err != nil {
		return err
	}
	return nil
}

// GetAllGeneralBots GeneralBotを全て取得
func (s *BotStoreImpl) GetAllGeneralBots() (arr []bot.GeneralBot, err error) {
	var gbs []*GeneralBotUser
	if err := db.Join("INNER", "general_bots", "general_bots.bot_user_id = users.id").Find(&gbs); err != nil {
		return nil, err
	}
	for _, v := range gbs {
		postURL, _ := url.Parse(v.GeneralBot.PostURL)
		arr = append(arr, bot.GeneralBot{
			ID:                uuid.Must(uuid.FromString(v.GeneralBot.ID)),
			BotUserID:         uuid.Must(uuid.FromString(v.GeneralBot.BotUserID)),
			Name:              v.User.Name,
			DisplayName:       v.User.DisplayName,
			Description:       v.GeneralBot.Description,
			IconFileID:        uuid.Must(uuid.FromString(v.User.Icon)),
			VerificationToken: v.GeneralBot.VerificationToken,
			AccessTokenID:     uuid.Must(uuid.FromString(v.GeneralBot.AccessTokenID)),
			PostURL:           *postURL,
			SubscribeEvents:   strings.Fields(v.GeneralBot.SubscribeEvents),
			Activated:         v.GeneralBot.Activated,
			IsValid:           v.GeneralBot.IsValid,
			OwnerID:           uuid.Must(uuid.FromString(v.GeneralBot.OwnerID)),
			CreatedAt:         v.GeneralBot.CreatedAt,
			UpdatedAt:         v.GeneralBot.UpdatedAt,
		})
	}
	return
}

// SaveWebhook Webhookをdbに保存
func (s *BotStoreImpl) SaveWebhook(w *bot.Webhook) error {
	u := &User{
		ID:          w.BotUserID.String(),
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(w.BotUserID.Bytes()),
		DisplayName: w.Name,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        w.IconFileID.String(),
		Status:      0, //TODO
		Bot:         true,
		BotType:     bot.TypeWebhook,
		Role:        role.Bot.ID(), //FIXME
	}
	wb := &WebhookBot{
		ID:          w.ID.String(),
		BotUserID:   w.BotUserID.String(),
		Description: w.Description,
		ChannelID:   w.ChannelID.String(),
		IsValid:     w.IsValid,
		CreatorID:   w.CreatorID.String(),
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}

	if _, err := db.UseBool().Insert(u, wb); err != nil {
		return err
	}
	return nil
}

// UpdateWebhook Webhookをdbに保存
func (s *BotStoreImpl) UpdateWebhook(w *bot.Webhook) error {
	if _, err := db.ID(w.BotUserID.String()).Update(&User{
		DisplayName: w.Name,
		Icon:        w.IconFileID.String(),
	}); err != nil {
		return err
	}
	if _, err := db.ID(w.ID.String()).UseBool("is_valid").Update(&WebhookBot{
		Description: w.Description,
		ChannelID:   w.ChannelID.String(),
		IsValid:     w.IsValid,
		CreatorID:   w.CreatorID.String(),
		UpdatedAt:   w.UpdatedAt,
	}); err != nil {
		return err
	}
	return nil
}

// GetAllWebhooks Webhookを全て取得
func (s *BotStoreImpl) GetAllWebhooks() (arr []bot.Webhook, err error) {
	var webhooks []*WebhookBotUser
	if err = db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Find(&webhooks); err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, bot.Webhook{
			ID:          uuid.Must(uuid.FromString(v.WebhookBot.ID)),
			BotUserID:   uuid.Must(uuid.FromString(v.WebhookBot.BotUserID)),
			Name:        v.User.DisplayName,
			Description: v.WebhookBot.Description,
			ChannelID:   uuid.Must(uuid.FromString(v.WebhookBot.ChannelID)),
			IconFileID:  uuid.Must(uuid.FromString(v.User.Icon)),
			CreatorID:   uuid.Must(uuid.FromString(v.WebhookBot.CreatorID)),
			CreatedAt:   v.WebhookBot.CreatedAt,
			UpdatedAt:   v.WebhookBot.UpdatedAt,
			IsValid:     v.WebhookBot.IsValid,
		})
	}
	return
}

// Plugin Plugin構造体
type Plugin struct {
	ID                string    `xorm:"char(36) not null pk"`
	BotUserID         string    `xorm:"char(36) not null unique"`
	Command           string    `xorm:"char(50) not null unique"`
	Description       string    `xorm:"text not null"`
	Usage             string    `xorm:"text not null"`
	VerificationToken string    `xorm:"text not null"`
	AccessTokenID     string    `xorm:"char(36) not null"`
	PostURL           string    `xorm:"text not null"`
	Activated         bool      `xorm:"bool not null"`
	IsValid           bool      `xorm:"bool not null"`
	CreatorID         string    `xorm:"char(36) not null"`
	CreatedAt         time.Time `xorm:"timestamp not null"`
	UpdatedAt         time.Time `xorm:"timestamp not null"`
}

// WebhookBot WebhookBot構造体
type WebhookBot struct {
	ID          string    `xorm:"char(36) not null pk"`
	BotUserID   string    `xorm:"char(36) not null unique"`
	Description string    `xorm:"text not null"`
	ChannelID   string    `xorm:"char(36) not null"`
	IsValid     bool      `xorm:"bool not null"`
	CreatorID   string    `xorm:"char(36) not null"`
	CreatedAt   time.Time `xorm:"timestamp not null"`
	UpdatedAt   time.Time `xorm:"timestamp not null"`
}

// GeneralBot GeneralBot構造体
type GeneralBot struct {
	ID                string    `xorm:"char(36) not null pk"`
	BotUserID         string    `xorm:"char(36) not null unique"`
	Description       string    `xorm:"text not null"`
	VerificationToken string    `xorm:"text not null"`
	AccessTokenID     string    `xorm:"char(36) not null"`
	PostURL           string    `xorm:"text not null"`
	SubscribeEvents   string    `xorm:"text not null"`
	Activated         bool      `xorm:"bool not null"`
	IsValid           bool      `xorm:"bool not null"`
	OwnerID           string    `xorm:"char(36) not null"`
	CreatedAt         time.Time `xorm:"timestamp not null"`
	UpdatedAt         time.Time `xorm:"timestamp not null"`
}

// BotInstalledChannel BotInstalledChannel構造体
type BotInstalledChannel struct {
	BotID       string `xorm:"char(36) not null unique(bot_channel)"`
	ChannelID   string `xorm:"char(36) not null unique(bot_channel)"`
	InstalledBy string `xorm:"char(36) not null"`
}

// BotOutgoingPostLog BotのPOST URLへのリクエストの結果のログ構造体
type BotOutgoingPostLog struct {
	RequestID  string `xorm:"char(36) not null pk"`
	BotUserID  string `xorm:"char(36) not null"`
	StatusCode int    `xorm:"int not null"`
	Request    string `xorm:"text not null"`
	Response   string `xorm:"text not null"`
	Error      string `xorm:"text not null"`
	Timestamp  int64  `xorm:"int64 not null"`
}

// TableName Pluginのテーブル名
func (*Plugin) TableName() string {
	return "plugins"
}

// TableName Webhookのテーブル名
func (*WebhookBot) TableName() string {
	return "webhook_bots"
}

// TableName GeneralBotのテーブル名
func (*GeneralBot) TableName() string {
	return "general_bots"
}

// TableName BotInstalledChannelのテーブル名
func (*BotInstalledChannel) TableName() string {
	return "bots_installed_channels"
}

// TableName BotOutgoingPostLogのテーブル名
func (*BotOutgoingPostLog) TableName() string {
	return "bot_outgoing_post_logs"
}

// WebhookBotUser WebhookBotUser構造体 内部にUser, WebhookBotを内包
type WebhookBotUser struct {
	*User       `xorm:"extends"`
	*WebhookBot `xorm:"extends"`
}

// TableName JOIN処理用
func (*WebhookBotUser) TableName() string {
	return "users"
}

// GeneralBotUser GeneralBotUser構造体 内部にUser, GeneralBotを内包
type GeneralBotUser struct {
	*User       `xorm:"extends"`
	*GeneralBot `xorm:"extends"`
}

// TableName JOIN処理用
func (*GeneralBotUser) TableName() string {
	return "users"
}

// PluginUser PluginUser構造体 内部にUser, Pluginを内包
type PluginUser struct {
	*User   `xorm:"extends"`
	*Plugin `xorm:"extends"`
}

// TableName JOIN処理用
func (*PluginUser) TableName() string {
	return "users"
}
