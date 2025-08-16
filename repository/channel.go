//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/set"
)

// ChangeChannelSubscriptionArgs チャンネル購読変更引数
type ChangeChannelSubscriptionArgs struct {
	Subscription map[uuid.UUID]model.ChannelSubscribeLevel
	KeepOffLevel bool
}

// UpdateChannelArgs チャンネル情報更新引数
type UpdateChannelArgs struct {
	UpdaterID          uuid.UUID
	Name               optional.Of[string]
	Topic              optional.Of[string]
	Visibility         optional.Of[bool]
	ForcedNotification optional.Of[bool]
	Parent             optional.Of[uuid.UUID]
}

// ChannelEventsQuery GetChannelEvents用クエリ
type ChannelEventsQuery struct {
	Channel   uuid.UUID
	Since     optional.Of[time.Time]
	Until     optional.Of[time.Time]
	Inclusive bool
	Limit     int
	Offset    int
	Asc       bool
}

// ChannelSubscriptionQuery GetChannelSubscriptions用クエリ
type ChannelSubscriptionQuery struct {
	UserID    optional.Of[uuid.UUID]
	ChannelID optional.Of[uuid.UUID]
	Level     model.ChannelSubscribeLevel
}

func (q ChannelSubscriptionQuery) SetUser(id uuid.UUID) ChannelSubscriptionQuery {
	q.UserID = optional.From(id)
	return q
}

func (q ChannelSubscriptionQuery) SetChannel(id uuid.UUID) ChannelSubscriptionQuery {
	q.ChannelID = optional.From(id)
	return q
}

func (q ChannelSubscriptionQuery) SetLevel(level model.ChannelSubscribeLevel) ChannelSubscriptionQuery {
	q.Level = level
	return q
}

// ChannelStats チャンネル統計情報
type ChannelStats struct {
	TotalMessageCount int64 `json:"totalMessageCount"`
	Stamps            []struct {
		ID    uuid.UUID `json:"id"`
		Count int64     `json:"count"`
		Total int64     `json:"total"`
	} `json:"stamps"`
	Users []struct {
		ID           uuid.UUID `json:"id"`
		MessageCount int64     `json:"messageCount"`
	} `json:"users"`
	DateTime time.Time `json:"datetime"`
}

// ChannelRepository チャンネルリポジトリ
type ChannelRepository interface {
	// GetPublicChannels 全ての公開チャンネルを返します
	GetPublicChannels() ([]*model.Channel, error)
	// CreateChannel チャンネルを作成します
	//
	// dmがtrueの場合、privateMembersに1人または2人のユーザーが入っている必要があります。
	CreateChannel(ch model.Channel, privateMembers set.UUID, dm bool) (*model.Channel, error)
	// UpdateChannel 指定したチャンネルの情報を変更します
	//
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	UpdateChannel(channelID uuid.UUID, args UpdateChannelArgs) (*model.Channel, error)
	// ArchiveChannels 指定したチャンネルをアーカイブします
	ArchiveChannels(ids []uuid.UUID) ([]*model.Channel, error)
	// GetChannel 指定したチャンネルを取得します
	//
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	GetChannel(channelID uuid.UUID) (*model.Channel, error)
	// GetDirectMessageChannel 指定したユーザー間のDMチャンネル取得します
	//
	// 存在しなかった場合、ErrNotFoundを返します。
	GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error)
	// GetDirectMessageChannelMapping 指定したユーザーのDMチャンネルのマッピングを取得します
	GetDirectMessageChannelMapping(userID uuid.UUID) ([]*model.DMChannelMapping, error)
	// GetDirectMessageChannelList 自分の参加しているDMチャンネルを新しい順にして取得します
	GetDirectMessageChannelList(userID uuid.UUID) ([]*model.DMChannelMapping, error)
	// GetPrivateChannelMemberIDs 指定したプライベートチャンネルのメンバーのUUIDを取得します
	GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// ChangeChannelSubscription ユーザーのチャンネルの購読を変更します
	//
	// channelIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないユーザーを指定した場合は無視されます。
	ChangeChannelSubscription(channelID uuid.UUID, args ChangeChannelSubscriptionArgs) (on []uuid.UUID, off []uuid.UUID, err error)
	// GetChannelSubscriptions 指定したクエリに基づいてチャンネル購読情報を取得します
	GetChannelSubscriptions(query ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error)
	// GetChannelEvents 指定したクエリでチャンネルイベントを取得します
	//
	// 負のoffset, limitは無視されます。
	// 指定した範囲内にlimitを超えてイベントが存在していた場合、trueを返します。
	GetChannelEvents(query ChannelEventsQuery) (events []*model.ChannelEvent, more bool, err error)
	// GetChannelStats 指定したチャンネルの統計情報を取得します
	//
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	GetChannelStats(channelID uuid.UUID, excludeDeletedMessages bool) (*ChannelStats, error)
	// RecordChannelEvent チャンネルイベントを記録します
	RecordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) error
}
