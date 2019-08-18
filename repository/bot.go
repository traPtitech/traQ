package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateBotArgs Bot情報更新引数
type UpdateBotArgs struct {
	DisplayName null.String
	Description null.String
	WebhookURL  null.String
	Privileged  null.Bool
	CreatorID   uuid.NullUUID
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
	// SetSubscribeEventsToBot 指定したBotの購読イベントを変更します
	//
	// 成功した場合、nilを返します。
	// 存在しないBotを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error
	// GetAllBots 全てのBotを取得します
	//
	// 成功した場合、Botの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllBots() ([]*model.Bot, error)
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
	// GetBotsByCreator 指定したCreatorのBotを全て取得します
	//
	// 成功した場合、Botの配列とnilを返します。
	// 存在しないユーザーの場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error)
	// GetBotsByChannel 指定したチャンネルに参加しているBotを全て取得します
	//
	// 成功した場合、Botの配列とnilを返します。
	// 存在しないチャンネルの場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error)
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
