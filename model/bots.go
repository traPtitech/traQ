package model

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/rbac/role"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BotStoreImpl Botデータ用ストアの実装
type BotStoreImpl struct {
	webhooks    sync.Map
	plugins     sync.Map
	generalBots sync.Map
	installed   sync.Map
}

// Init BotStoreを初期化。必ず呼ぶこと
func (s *BotStoreImpl) Init() error {
	var webhooks []*WebhookBotUser
	if err := db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Find(&webhooks); err != nil {
		return err
	}
	for _, v := range webhooks {
		id := uuid.Must(uuid.FromString(v.WebhookBot.ID))
		s.webhooks.Store(id, bot.Webhook{
			ID:          id,
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

	var gbs []*GeneralBotUser
	if err := db.Join("INNER", "general_bots", "general_bots.bot_user_id = users.id").Find(&gbs); err != nil {
		return err
	}
	for _, v := range gbs {
		id := uuid.Must(uuid.FromString(v.GeneralBot.ID))
		postURL, _ := url.Parse(v.GeneralBot.PostURL)
		s.generalBots.Store(id, bot.GeneralBot{
			ID:                id,
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

		var bics []BotInstalledChannel
		if err := db.Where("bot_id = ?", v.GeneralBot.ID).Find(&bics); err != nil {
			return err
		}
		for _, v := range bics {
			s.installed.Store(bot.InstalledChannel{
				BotID:       uuid.Must(uuid.FromString(v.BotID)),
				ChannelID:   uuid.Must(uuid.FromString(v.ChannelID)),
				InstalledBy: uuid.Must(uuid.FromString(v.InstalledBy)),
			}, struct{}{})
		}
	}

	return nil
}

func (s *BotStoreImpl) SaveGeneralBot(b *bot.GeneralBot) error {
	_, ok := s.GetGeneralBot(b.ID)
	if !ok {
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
		s.generalBots.Store(b.ID, *b)
	} else {
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
		s.generalBots.Store(b.ID, *b)
	}

	return nil
}

func (s *BotStoreImpl) GetInstalledChannels(botID uuid.UUID) (arr []bot.InstalledChannel) {
	s.installed.Range(func(value, _ interface{}) bool {
		ic := value.(bot.InstalledChannel)
		if ic.BotID == botID {
			arr = append(arr, ic)
		}
		return true
	})
	return
}

func (s *BotStoreImpl) GetInstalledBot(channelID uuid.UUID) (arr []bot.InstalledChannel) {
	s.installed.Range(func(value, _ interface{}) bool {
		ic := value.(bot.InstalledChannel)
		if ic.ChannelID == channelID {
			arr = append(arr, ic)
		}
		return true
	})
	return
}

func (s *BotStoreImpl) InstallBot(botID, channelID, userID uuid.UUID) error {
	bic := &BotInstalledChannel{
		BotID:       botID.String(),
		ChannelID:   channelID.String(),
		InstalledBy: userID.String(),
	}

	if _, err := db.InsertOne(bic); err != nil {
		return err
	}
	s.installed.Store(bot.InstalledChannel{
		BotID:       botID,
		ChannelID:   channelID,
		InstalledBy: userID,
	}, struct{}{})
	return nil
}

func (s *BotStoreImpl) UninstallBot(botID, channelID uuid.UUID) error {
	var uid uuid.UUID
	s.installed.Range(func(value, _ interface{}) bool {
		v := value.(bot.InstalledChannel)
		if v.BotID == botID && v.ChannelID == channelID {
			uid = v.InstalledBy
			return false
		}
		return true
	})

	if uid == uuid.Nil {
		return nil
	}

	bic := &BotInstalledChannel{
		BotID:     botID.String(),
		ChannelID: channelID.String(),
	}

	if _, err := db.Delete(bic); err != nil {
		return err
	}
	s.installed.Delete(bot.InstalledChannel{
		BotID:       botID,
		ChannelID:   channelID,
		InstalledBy: uid,
	})
	return nil
}

// GetAllGeneralBots GeneralBotを全て取得
func (s *BotStoreImpl) GetAllGeneralBots() (arr []bot.GeneralBot) {
	s.generalBots.Range(func(key, value interface{}) bool {
		arr = append(arr, value.(bot.GeneralBot))
		return true
	})
	return
}

// GetGeneralBot GeneralBotを取得
func (s *BotStoreImpl) GetGeneralBot(id uuid.UUID) (bot.GeneralBot, bool) {
	b, ok := s.generalBots.Load(id)
	if !ok {
		return bot.GeneralBot{}, false
	}
	return b.(bot.GeneralBot), true
}

// SaveWebhook Webhookをdbに保存
func (s *BotStoreImpl) SaveWebhook(w *bot.Webhook) error {
	_, ok := s.GetWebhook(w.ID)
	if !ok {
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
		s.webhooks.Store(w.ID, *w)
	} else {
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
		s.webhooks.Store(w.ID, *w)
	}

	return nil
}

// GetAllWebhooks Webhookを全て取得
func (s *BotStoreImpl) GetAllWebhooks() (arr []bot.Webhook) {
	s.webhooks.Range(func(key, value interface{}) bool {
		arr = append(arr, value.(bot.Webhook))
		return true
	})
	return
}

// GetWebhook Webhookを取得
func (s *BotStoreImpl) GetWebhook(id uuid.UUID) (bot.Webhook, bool) {
	w, ok := s.webhooks.Load(id)
	if !ok {
		return bot.Webhook{}, false
	}
	return w.(bot.Webhook), true
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
	ID         string    `xorm:"char(36) not null pk"`
	BotUserID  string    `xorm:"char(36) not null"`
	StatusCode int       `xorm:"int not null"`
	Summary    string    `xorm:"text not null"`
	DateTime   time.Time `xorm:"timestamp not null"`
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
