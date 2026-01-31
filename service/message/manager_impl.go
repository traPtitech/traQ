package message

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/motoki317/sc"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/optional"
)

const (
	cacheSize = 512
	cacheTTL  = time.Minute
)

const PinLimit = 100 // ピン留めの上限数

type manager struct {
	CM channel.Manager
	R  repository.Repository
	L  *zap.Logger
	P  sync.WaitGroup

	cache *sc.Cache[uuid.UUID, *message]
}

func NewMessageManager(repo repository.Repository, cm channel.Manager, logger *zap.Logger) (Manager, error) {
	return &manager{
		CM: cm,
		R:  repo,
		L:  logger.Named("message_manager"),
		cache: sc.NewMust(func(ctx context.Context, key uuid.UUID) (*message, error) {
			m, err := repo.GetMessageByID(ctx, key)
			if err != nil {
				if err == repository.ErrNotFound {
					return nil, ErrNotFound
				}
				return nil, fmt.Errorf("failed to GetMessageByID: %w", err)
			}
			return &message{Model: m}, nil
		}, cacheTTL, cacheTTL*2, sc.With2QBackend(cacheSize)),
	}, nil
}

func (m *manager) Get(ctx context.Context, id uuid.UUID) (Message, error) {
	return m.get(ctx, id)
}

func (m *manager) get(ctx context.Context, id uuid.UUID) (*message, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}

	// メモリキャッシュから取得。キャッシュに無い場合はキャッシュの replaceFn で自動取得し、キャッシュに追加
	return m.cache.Get(ctx, id)
}

func (m *manager) GetIn(ctx context.Context, ids []uuid.UUID) ([]Message, error) {
	messages, _, err := m.R.GetMessages(ctx, repository.MessagesQuery{IDIn: optional.From(ids)})
	if err != nil {
		return nil, err
	}
	ret := utils.Map(messages, func(m *model.Message) Message {
		return &message{Model: m}
	})
	return ret, nil
}

func (m *manager) GetTimeline(ctx context.Context, query TimelineQuery) (Timeline, error) {
	q := repository.MessagesQuery{
		User:                     query.User,
		Channel:                  query.Channel,
		ChannelsSubscribedByUser: query.ChannelsSubscribedByUser,
		Since:                    query.Since,
		Until:                    query.Until,
		Inclusive:                query.Inclusive,
		Limit:                    query.Limit,
		Offset:                   query.Offset,
		Asc:                      query.Asc,
		ExcludeDMs:               query.ExcludeDMs,
		DisablePreload:           query.DisablePreload,
	}
	messages, more, err := m.R.GetMessages(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to GetMessages: %w", err)
	}

	return &timeline{
		query:       query,
		records:     messages,
		more:        more,
		preloaded:   !q.DisablePreload,
		retrievedAt: time.Now(),
		man:         m,
	}, nil
}

func (m *manager) CreateDM(ctx context.Context, from, to uuid.UUID, content string) (Message, error) {
	// DMチャンネルを取得
	ch, err := m.CM.GetDMChannel(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return m.create(ctx, ch.ID, from, content)
}

func (m *manager) Create(ctx context.Context, channelID, userID uuid.UUID, content string) (Message, error) {
	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(ctx, channelID) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(channelID) {
		return nil, ErrChannelArchived
	}

	return m.create(ctx, channelID, userID, content)
}

func (m *manager) create(ctx context.Context, channelID, userID uuid.UUID, content string) (Message, error) {
	// 作成
	msg, err := m.R.CreateMessage(ctx, userID, channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to CreateMessage: %w", err)
	}
	return &message{Model: msg}, nil
}

func (m *manager) Edit(ctx context.Context, id uuid.UUID, content string) error {
	// メッセージ取得
	msg, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return ErrChannelArchived
	}

	// 更新
	if err := m.R.UpdateMessage(ctx, id, content); err != nil {
		switch err {
		case repository.ErrNotFound:
			return ErrNotFound
		default:
			return fmt.Errorf("failed to UpdateMessage: %w", err)
		}
	}
	m.cache.Forget(id)

	return nil
}

func (m *manager) Delete(ctx context.Context, id uuid.UUID) error {
	// メッセージ取得
	msg, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return ErrChannelArchived
	}

	// 削除
	if err := m.R.DeleteMessage(ctx, id); err != nil {
		switch err {
		case repository.ErrNotFound:
			return ErrNotFound
		default:
			return fmt.Errorf("failed to DeleteMessage: %w", err)
		}
	}
	m.cache.Forget(id)

	return nil
}

func (m *manager) Pin(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Pin, error) {
	// メッセージ取得
	msg, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// すでにピンされているか
	if msg.GetPin() != nil {
		return nil, ErrAlreadyExists
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return nil, ErrChannelArchived
	}

	// チャンネルに上限数以上のメッセージがピン留めされていないか確認
	pins, err := m.R.GetPinnedMessageByChannelID(ctx, msg.GetChannelID())
	if err != nil {
		return nil, err
	}
	if len(pins) >= PinLimit {
		return nil, ErrPinLimitExceeded
	}

	// ピン
	pin, err := m.R.PinMessage(ctx, id, userID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return nil, ErrNotFound
		case repository.ErrAlreadyExists:
			return nil, ErrAlreadyExists
		default:
			return nil, fmt.Errorf("failed to PinMessage: %w", err)
		}
	}
	m.cache.Forget(id)

	// ロギング
	m.recordChannelEvent(pin.Message.ChannelID, model.ChannelEventPinAdded, model.ChannelEventDetail{
		"userId":    userID,
		"messageId": pin.MessageID,
	}, pin.CreatedAt)
	return pin, nil
}

func (m *manager) Unpin(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// メッセージ取得
	msg, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// ピンがあるかどうか
	if msg.GetPin() == nil {
		return ErrNotFound
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return ErrChannelArchived
	}

	// ピン外し
	pin, err := m.R.UnpinMessage(ctx, id)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return ErrNotFound
		default:
			return fmt.Errorf("failed to UnpinMessage: %w", err)
		}
	}
	m.cache.Forget(id)

	// ロギング
	m.recordChannelEvent(pin.Message.ChannelID, model.ChannelEventPinRemoved, model.ChannelEventDetail{
		"userId":    userID,
		"messageId": pin.MessageID,
	}, time.Now())
	return nil
}

func (m *manager) AddStamps(ctx context.Context, id, stampID, userID uuid.UUID, n int) (*model.MessageStamp, error) {
	// メッセージ取得
	msg, err := m.get(ctx, id)
	if err != nil {
		return nil, err
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return nil, ErrChannelArchived
	}

	// スタンプを押す
	ms, err := m.R.AddStampToMessage(ctx, id, stampID, userID, n)
	if err != nil {
		return nil, fmt.Errorf("failed to AddStampToMessage: %w", err)
	}

	// キャッシュ削除
	m.cache.Forget(id)

	return ms, nil
}

func (m *manager) RemoveStamps(ctx context.Context, id, stampID, userID uuid.UUID) error {
	// メッセージ取得
	msg, err := m.get(ctx, id)
	if err != nil {
		return err
	}

	// チャンネルがアーカイブされているかどうか確認
	if m.CM.IsPublicChannel(context.Background(), msg.GetChannelID()) && m.CM.PublicChannelTree(context.Background()).IsArchivedChannel(msg.GetChannelID()) {
		return ErrChannelArchived
	}

	// スタンプを消す
	if err := m.R.RemoveStampFromMessage(ctx, id, stampID, userID); err != nil {
		return fmt.Errorf("failed to RemoveStampFromMessage: %w", err)
	}

	// キャッシュ削除
	m.cache.Forget(id)

	return nil
}

func (m *manager) Wait(_ context.Context) error {
	m.P.Wait()
	return nil
}

func (m *manager) recordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) {
	m.P.Add(1)
	go func() {
		defer m.P.Done()

		err := m.R.RecordChannelEvent(context.Background(), channelID, eventType, detail, datetime)
		if err != nil {
			m.L.Warn("failed to record channel event", zap.Error(err), zap.Stringer("channelID", channelID), zap.Stringer("type", eventType), zap.Any("detail", detail), zap.Time("datetime", datetime))
		}
	}()
}
