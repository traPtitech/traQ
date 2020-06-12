//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// UpdateBotArgs Bot情報更新引数
type UpdateBotArgs struct {
	DisplayName     optional.String
	Description     optional.String
	WebhookURL      optional.String
	Privileged      optional.Bool
	CreatorID       optional.UUID
	SubscribeEvents model.BotEventTypes
}

// BotsQuery Bot情報取得用クエリ
type BotsQuery struct {
	IsPrivileged    optional.Bool
	IsActive        optional.Bool
	IsCMemberOf     optional.UUID
	SubscribeEvents model.BotEventTypes
	Creator         optional.UUID
	ID              optional.UUID
	UserID          optional.UUID
}

// Privileged 特権Botである
func (q BotsQuery) Privileged() BotsQuery {
	q.IsPrivileged = optional.BoolFrom(true)
	return q
}

// Active 有効である
func (q BotsQuery) Active() BotsQuery {
	q.IsActive = optional.BoolFrom(true)
	return q
}

// CreatedBy userIDによって作成された
func (q BotsQuery) CreatedBy(userID uuid.UUID) BotsQuery {
	q.Creator = optional.UUIDFrom(userID)
	return q
}

// CMemberOf channelIDに入っている
func (q BotsQuery) CMemberOf(channelID uuid.UUID) BotsQuery {
	q.IsCMemberOf = optional.UUIDFrom(channelID)
	return q
}

// Subscribe eventsを購読している
func (q BotsQuery) Subscribe(events ...model.BotEventType) BotsQuery {
	if q.SubscribeEvents == nil {
		q.SubscribeEvents = model.BotEventTypes{}
	} else {
		q.SubscribeEvents = q.SubscribeEvents.Clone()
	}
	for _, event := range events {
		q.SubscribeEvents[event] = struct{}{}
	}
	return q
}

// BotID 指定したIDのBotである
func (q BotsQuery) BotID(id uuid.UUID) BotsQuery {
	q.ID = optional.UUIDFrom(id)
	return q
}

// BotUserID 指定したユーザーIDのBotである
func (q BotsQuery) BotUserID(id uuid.UUID) BotsQuery {
	q.ID = optional.UUIDFrom(id)
	return q
}

// BotRepository Botリポジトリ
type BotRepository interface {
	// CreateBot Botを作成します
	//
	// 成功した場合、Botとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// nameが既に使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error)
	// UpdateBot 指定したBotの情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないBotを指定した場合、ErrNotFoundを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateBot(id uuid.UUID, args UpdateBotArgs) error
	// GetAllBots 全てのBotを取得します
	//
	// 成功した場合、Botの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetBots(query BotsQuery) ([]*model.Bot, error)
	// GetBotByID 指定したIDのBotを取得します
	//
	// 成功した場合、Botとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetBotByID(id uuid.UUID) (*model.Bot, error)
	// GetBotByBotUserID 指定したユーザーIDのBotを取得します
	//
	// 成功した場合、Botとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetBotByBotUserID(id uuid.UUID) (*model.Bot, error)
	// GetBotByCode 指定したBotCodeのBotを取得します
	//
	// 成功した場合、Botとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetBotByCode(code string) (*model.Bot, error)
	// ChangeBotState Botの状態を変更します
	//
	// 成功した場合、nilを返します。
	// 存在しないBotを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeBotState(id uuid.UUID, state model.BotState) error
	// ReissueBotTokens 指定したBotの各種トークンを再発行します
	//
	// 成功した場合、Botとnilを返します。
	// 存在しないBotを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ReissueBotTokens(id uuid.UUID) (*model.Bot, error)
	// DeleteBot 指定したBotを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しないBotを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteBot(id uuid.UUID) error
	// AddBotToChannel 指定したBotをチャンネルに参加させます
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddBotToChannel(botID, channelID uuid.UUID) error
	// RemoveBotFromChannel 指定したBotを指定したチャンネルから退出させます
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveBotFromChannel(botID, channelID uuid.UUID) error
	// GetParticipatingChannelIDsByBot 指定したBotが参加しているチャンネルのIDを取得します
	//
	// 成功した場合、チャンネルUUIDの配列とnilを返します。
	// 存在しないBotを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error)
	// WriteBotEventLog Botイベントログを書き込みます
	//
	// 成功した場合、nilを返します。
	// DBによるエラーを返すことがあります。
	WriteBotEventLog(log *model.BotEventLog) error
	// GetBotEventLogs 指定したBotのイベントログを取得します
	//
	// 成功した場合、イベントログの配列とnilを返します。負のoffset, limitは無視されます。
	// 存在しないBotを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error)
}
