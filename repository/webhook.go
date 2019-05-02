package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateWebhookArgs Webhook情報更新引数
type UpdateWebhookArgs struct {
	Name        null.String
	Description null.String
	ChannelID   uuid.NullUUID
	Secret      null.String
}

// WebhookRepository Webhookボットリポジトリ
type WebhookRepository interface {
	// CreateWebhook Webhookを作成します
	//
	// 成功した場合、Webhookとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	CreateWebhook(name, description string, channelID, creatorID uuid.UUID, secret string) (model.Webhook, error)
	// UpdateWebhook Webhookを更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないWebhookの場合、ErrNotFoundを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateWebhook(id uuid.UUID, args UpdateWebhookArgs) error
	// DeleteWebhook Webhookを削除します
	//
	// 成功した場合、nilを返します。WebhookのUserはstatusがdeactivatedになります。
	// 既に存在しなかった場合、ErrNotFoundを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteWebhook(id uuid.UUID) error
	// GetWebhookByBotUserID 指定したWebhookを取得します
	//
	// 成功した場合、Webhookとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetWebhook(id uuid.UUID) (model.Webhook, error)
	// GetWebhookByBotUserID 指定したユーザーUUIDをもつWebhookを取得します
	//
	// 成功した場合、Webhookとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetWebhookByBotUserID(id uuid.UUID) (model.Webhook, error)
	// GetAllWebhooks Webhookを全て取得します
	//
	// 成功した場合、Webhookの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllWebhooks() ([]model.Webhook, error)
	// GetWebhooksByCreator 指定した制作者のWebhookを全て取得します
	//
	// 成功した場合、Webhookの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error)
}
