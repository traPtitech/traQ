package model

import (
	"encoding/base64"
	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

// Webhook : Webhook構造体
type Webhook struct {
	ID        string `xorm:"char(36) not null pk"`
	UserID    string `xorm:"char(36) not null"`
	ChannelID string `xorm:"char(36) not null"`
}

// WebhookBotUser : WebhookBotUser構造体 内部にBot, Webhook, Userを内包
type WebhookBotUser struct {
	*Bot     `xorm:"extends"`
	*User    `xorm:"extends"`
	*Webhook `xorm:"extends"`
}

// TableName : Webhookのテーブル名
func (*Webhook) TableName() string {
	return "webhooks"
}

// TableName : JOIN処理用
func (*WebhookBotUser) TableName() string {
	return "users"
}

// UpdateChannelID : デフォルトの投稿先チャンネルを変更します。
func (w *Webhook) UpdateChannelID(channelID string) error {
	w.ChannelID = channelID
	_, err := db.ID(w.ID).Update(w)
	return err
}

func getWebhookJoinedDB() *xorm.Session {
	return db.Join("INNER", "bots", "bots.user_id = users.id").Join("INNER", "webhooks", "webhooks.user_id = users.id")
}

// GetWebhook : Webhookを取得します。
func GetWebhook(webhookID string) (*WebhookBotUser, error) {
	webhook := &WebhookBotUser{}

	has, err := getWebhookJoinedDB().Where("webhooks.id = ?", webhookID).Get(webhook)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}

	return webhook, nil
}

// GetWebhooksByCreator : 指定したユーザーが作成したWebhookの一覧を取得します。
func GetWebhooksByCreator(userID string) (webhooks []*WebhookBotUser, err error) {
	err = getWebhookJoinedDB().Where("bots.creator_id = ?", userID).Find(&webhooks)
	return
}

// GetAllWebhooks : 全てのWebhookの一覧を取得します。
func GetAllWebhooks() (webhooks []*WebhookBotUser, err error) {
	err = getWebhookJoinedDB().Find(&webhooks)
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
		ID:        CreateUUID(),
		UserID:    user.ID,
		ChannelID: channelID,
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
