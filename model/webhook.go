package model

import (
	"encoding/base64"
	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

// Webhook : Webhook構造体
type Webhook struct {
	UserID    string `xorm:"char(36) not null pk"` //webhookIDと同義
	Token     string `xorm:"varchar(32) not null"`
	ChannelID string `xorm:"char(36) not null"`
}

// WebhookBotUser : WebhookBotUser構造体 内部にBot, Webhook, Userを内包
type WebhookBotUser struct {
	*Bot     `xorm:"extends"`
	*Webhook `xorm:"extends"`
	*User    `xorm:"extends"`
}

// TableName : Webhookのテーブル名
func (*Webhook) TableName() string {
	return "webhooks"
}

// TableName : JOIN処理用
func (*WebhookBotUser) TableName() string {
	return "users"
}

func getWebhookJoinedDB() *xorm.Session {
	return db.Join("INNER", "bots", "bots.user_id = users.id").Join("INNER", "webhooks", "webhooks.user_id = users.id")
}

// GetWebhook : Webhookを取得します。
func GetWebhook(webhookID string) (*WebhookBotUser, error) {
	webhook := &WebhookBotUser{}

	has, err := getWebhookJoinedDB().Where("users.id = ?", webhookID).Get(webhook)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}

	return webhook, nil
}

// GetWebhooksByChannelID : 指定したchannelにあるWebhookを全て取得します
func GetWebhooksByChannelID(channelID string) (webhooks []*WebhookBotUser, err error) {
	err = getWebhookJoinedDB().Where("webhooks.channel_id = ?", channelID).Find(&webhooks)
	return
}

// CreateWebhook : Webhookを作成します。
func CreateWebhook(name, description, channelID, creatorID, iconFileID string) (*WebhookBotUser, error) {
	if len(name) == 0 || len(name) > 32 {
		return nil, ErrBotInvalidName
	}
	if len(description) == 0 {
		return nil, ErrBotRequireDescription
	}

	botUID := uuid.NewV4()
	user := &User{
		ID:          botUID.String(),
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(botUID.Bytes()),
		DisplayName: name,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        "",
		Status:      1, //TODO
		Bot:         true,
	}

	//iconがなければ生成
	if len(iconFileID) != 36 {
		fileID, err := generateIcon(user.Name, serverUser.ID)
		if err != nil {
			return nil, err
		}
		user.Icon = fileID
	} else {
		user.Icon = iconFileID
	}

	if _, err := db.Insert(user); err != nil {
		return nil, err
	}

	bot := &Bot{
		UserID:      user.ID,
		Type:        BotTypeWebhook,
		Description: description,
		IsValid:     true,
		CreatorID:   creatorID,
		UpdaterID:   creatorID,
	}
	webhook := &Webhook{
		UserID:    user.ID,
		ChannelID: channelID,
		Token:     base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
	}

	if _, err := db.Insert(bot, webhook); err != nil {
		return nil, err
	}

	return &WebhookBotUser{
		Bot:     bot,
		Webhook: webhook,
		User:    user,
	}, nil
}
