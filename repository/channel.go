package repository

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
	"time"
)

var (
	// ErrChannelDepthLimitation チャンネルの深さ制限を超えている
	ErrChannelDepthLimitation = errors.New("channel depth limit exceeded")
)

// ChangeChannelSubscriptionArgs チャンネル購読変更引数
type ChangeChannelSubscriptionArgs struct {
	UpdaterID    uuid.UUID
	Subscription map[uuid.UUID]model.ChannelSubscribeLevel
	KeepOffLevel bool
}

// UpdateChannelArgs チャンネル情報更新引数
type UpdateChannelArgs struct {
	UpdaterID          uuid.UUID
	Name               null.String
	Topic              null.String
	Visibility         null.Bool
	ForcedNotification null.Bool
	Parent             uuid.NullUUID
}

// ChannelEventsQuery GetChannelEvents用クエリ
type ChannelEventsQuery struct {
	Channel   uuid.UUID
	Since     null.Time
	Until     null.Time
	Inclusive bool
	Limit     int
	Offset    int
	Asc       bool
}

// ChannelSubscriptionQuery GetChannelSubscriptions用クエリ
type ChannelSubscriptionQuery struct {
	UserID    uuid.NullUUID
	ChannelID uuid.NullUUID
	Level     model.ChannelSubscribeLevel
}

func (q ChannelSubscriptionQuery) SetUser(id uuid.UUID) ChannelSubscriptionQuery {
	q.UserID = uuid.NullUUID{Valid: true, UUID: id}
	return q
}

func (q ChannelSubscriptionQuery) SetChannel(id uuid.UUID) ChannelSubscriptionQuery {
	q.ChannelID = uuid.NullUUID{Valid: true, UUID: id}
	return q
}

func (q ChannelSubscriptionQuery) SetLevel(level model.ChannelSubscribeLevel) ChannelSubscriptionQuery {
	q.Level = level
	return q
}

// ChannelStats チャンネル統計情報
type ChannelStats struct {
	TotalMessageCount int       `json:"totalMessageCount"`
	DateTime          time.Time `json:"datetime"`
}

// ChannelRepository チャンネルリポジトリ
type ChannelRepository interface {
	// CreatePublicChannel パブリックチャンネルを作成します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 作成不可能な親チャンネルを指定した場合、ErrForbiddenを返します。
	// 階層数制限に到達する場合、ErrChannelDepthLimitationを返します。
	// 存在しない親チャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error)
	// UpdateChannel 指定したチャンネルの情報を変更します
	//
	// 成功した場合、nilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 変更不可能なチャンネルを指定した場合、ErrForbiddenを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// 階層数制限に到達する場合、ErrChannelDepthLimitationを返します。
	// DBによるエラーを返すことがあります。
	UpdateChannel(channelID uuid.UUID, args UpdateChannelArgs) error
	// GetChannel 指定したチャンネルを取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannel(channelID uuid.UUID) (*model.Channel, error)
	// GetChannelByMessageID 指定したメッセージの投稿先チャンネルを取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error)
	// GetChannelsByUserID 指定したユーザーがアクセス可能なチャンネルを全て取得します
	//
	// 成功した場合、チャンネルの配列とnilを返します。
	// 存在しないユーザーやuuid.Nilを指定した場合、パブリックチャンネルのみを返します。
	// DBによるエラーを返すことがあります。
	GetChannelsByUserID(userID uuid.UUID) ([]*model.Channel, error)
	// GetDirectMessageChannel 引数に指定したユーザー間のDMチャンネル取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error)
	// GetDirectMessageChannelMapping 引数に指定したユーザーのDMチャンネルのチャンネルUUID->ユーザーUUIDのマッピングを取得します
	//
	// 成功した場合、マッピングとnilを返します。
	// DBによるエラーを返すことがあります。
	GetDirectMessageChannelMapping(userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	// IsChannelAccessibleToUser 指定したチャンネルが指定したユーザーからアクセス可能かどうかを返します
	//
	// アクセス可能な場合、trueとnilを返します。
	// 存在しないチャンネル・ユーザーを指定した場合、falseとnilを返します。
	// DBによるエラーを返すことがあります。
	IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error)
	// GetChildrenChannelIDs 指定したチャンネルの子チャンネルのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetChildrenChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// GetPrivateChannelMemberIDs 指定したプライベートチャンネルのメンバーのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// ChangeChannelSubscription ユーザーのチャンネルの購読を変更します
	//
	// 成功した場合、nilを返します。
	// channelIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないユーザーを指定した場合は無視されます。
	// DBによるエラーを返すことがあります。
	ChangeChannelSubscription(channelID uuid.UUID, args ChangeChannelSubscriptionArgs) error
	// GetChannelSubscriptions 指定したクエリに基づいてチャンネル購読情報を取得します
	//
	// 成功した場合、購読情報の配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetChannelSubscriptions(query ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error)
	// GetChannelEvents 指定したクエリでチャンネルイベントを取得します
	//
	// 成功した場合、イベントの配列を返します。負のoffset, limitは無視されます。
	// 指定した範囲内にlimitを超えてイベントが存在していた場合、trueを返します。
	// DBによるエラーを返すことがあります。
	GetChannelEvents(query ChannelEventsQuery) (events []*model.ChannelEvent, more bool, err error)
	// GetChannelStats 指定したチャンネルの統計情報を取得します
	//
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannelStats(channelID uuid.UUID) (*ChannelStats, error)
	// GetChannelTree チャンネルツリーを取得します
	GetChannelTree() ChannelTree
}
