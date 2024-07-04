package channel

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/set"
	"github.com/traPtitech/traQ/utils/validator"
)

var (
	dmChannelRootUUID  = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))
	pubChannelRootUUID = uuid.Nil
)

type managerImpl struct {
	R repository.ChannelRepository
	L *zap.Logger
	T *treeImpl
	P sync.WaitGroup

	MaxChannelDepth int
}

func InitChannelManager(repo repository.ChannelRepository, logger *zap.Logger) (Manager, error) {
	channels, err := repo.GetPublicChannels()
	if err != nil {
		return nil, fmt.Errorf("failed to init channel.Manager: %w", err)
	}

	m := &managerImpl{
		R:               repo,
		L:               logger.Named("channel_manager"),
		MaxChannelDepth: 5,
	}
	m.T, err = makeChannelTree(channels)
	if err != nil {
		return nil, fmt.Errorf("failed to init channel.Manager: %w", err)
	}

	return m, nil
}

func (m *managerImpl) GetChannel(id uuid.UUID) (*model.Channel, error) {
	ch, err := m.T.GetModel(id)
	if err == nil {
		return ch, nil
	}

	ch, err = m.R.GetChannel(id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to GetChannel: %w", err)
	}
	ch.ChildrenID = make([]uuid.UUID, 0)
	return ch, nil
}

func (m *managerImpl) GetChannelPathFromID(id uuid.UUID) string {
	return m.T.getChannelPath(id)
}

func (m *managerImpl) GetChannelFromPath(path string) (*model.Channel, error) {
	id :=  m.T.getChannelIDFromPath(path)
	if id == uuid.Nil {
		return nil, ErrInvalidChannelPath
	}
	return m.GetChannel(id)
}

func (m *managerImpl) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	m.T.Lock()
	defer m.T.Unlock()

	// チャンネル名の制約を確認
	if !validator.ChannelRegex.MatchString(name) {
		return nil, ErrInvalidChannelName
	}

	// チャンネル名の重複を確認
	if m.T.isChildPresent(name, parent) {
		return nil, ErrChannelNameConflicts
	}

	if parent != pubChannelRootUUID {
		// 親チャンネルの存在を確認
		if !m.T.isChannelPresent(parent) {
			return nil, ErrInvalidParentChannel
		}
		// 親チャンネルがアーカイブされているかどうか確認
		if m.T.isArchivedChannel(parent) {
			return nil, ErrChannelArchived
		}
		// 深さを検証
		if len(m.T.getAscendantIDs(parent))+2 > m.MaxChannelDepth {
			return nil, ErrTooDeepChannel
		}
	}

	// チャンネル作成
	ch, err := m.R.CreateChannel(model.Channel{
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsForced:  false,
		IsVisible: true,
	}, nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to CreateChannel: %w", err)
	}
	m.T.add(ch)
	if parent != pubChannelRootUUID {
		// ロギング
		m.recordChannelEvent(ch.ParentID, model.ChannelEventChildCreated, model.ChannelEventDetail{
			"userId":    ch.CreatorID,
			"channelId": ch.ID,
		}, ch.CreatedAt)
	}
	m.L.Info(fmt.Sprintf("channel #%s was created", m.T.getChannelPath(ch.ID)), zap.Stringer("cid", ch.ID))
	return ch, nil
}

func (m *managerImpl) UpdateChannel(id uuid.UUID, args repository.UpdateChannelArgs) error {
	ch, err := m.GetChannel(id)
	if err != nil {
		return ErrChannelNotFound
	}

	// トピックが同じだった場合、トピックの引数自体を無効化
	if ch.Topic == args.Topic.V {
		args.Topic = optional.New("", false)
	}

	m.T.Lock()
	defer m.T.Unlock()

	eventRecords := map[model.ChannelEventType]model.ChannelEventDetail{}
	if args.Topic.Valid {
		if ch.IsArchived() {
			return ErrChannelArchived
		}
		eventRecords[model.ChannelEventTopicChanged] = model.ChannelEventDetail{
			"userId": args.UpdaterID,
			"before": ch.Topic,
			"after":  args.Topic.V,
		}
	}
	if args.Visibility.Valid && ch.IsVisible != args.Visibility.V {
		eventRecords[model.ChannelEventVisibilityChanged] = model.ChannelEventDetail{
			"userId":     args.UpdaterID,
			"visibility": args.Visibility.V,
		}
	}
	if args.ForcedNotification.Valid && ch.IsForced != args.ForcedNotification.V {
		eventRecords[model.ChannelEventForcedNotificationChanged] = model.ChannelEventDetail{
			"userId": args.UpdaterID,
			"force":  args.ForcedNotification.V,
		}
	}
	if args.Name.Valid || args.Parent.Valid {
		// チャンネル名重複を確認
		{
			var (
				n string
				p uuid.UUID
			)

			if args.Name.Valid {
				n = args.Name.V
			} else {
				n = ch.Name
			}
			if args.Parent.Valid {
				p = args.Parent.V
			} else {
				p = ch.ParentID
			}

			if m.T.isChildPresent(n, p) {
				return ErrChannelNameConflicts
			}
		}

		if args.Name.Valid {
			// チャンネル名検証
			if !validator.ChannelRegex.MatchString(args.Name.V) {
				return ErrInvalidChannelName
			}
			eventRecords[model.ChannelEventNameChanged] = model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"before": ch.Name,
				"after":  args.Name.V,
			}
		}
		if args.Parent.Valid {
			if args.Parent.V != pubChannelRootUUID {
				// 親チャンネル検証
				if !m.T.isChannelPresent(args.Parent.V) {
					return ErrInvalidParentChannel
				}

				// archiveされていないチャンネルを、アーカイブされているチャンネルの傘下に移動はできない
				if !ch.IsArchived() && m.T.isArchivedChannel(args.Parent.V) {
					return ErrInvalidParentChannel
				}

				// 深さを検証
				ascs := append(m.T.getAscendantIDs(args.Parent.V), args.Parent.V)
				for _, id := range ascs {
					if id == ch.ID {
						return ErrTooDeepChannel // ループ検出
					}
				}
				// 親チャンネル + 自分を含めた子チャンネルの深さ
				if len(ascs)+m.T.getChannelDepth(ch.ID) > m.MaxChannelDepth {
					return ErrTooDeepChannel
				}
			}
			eventRecords[model.ChannelEventParentChanged] = model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"before": ch.ParentID,
				"after":  args.Parent.V,
			}
		}
	}

	ch, err = m.R.UpdateChannel(id, args)
	if err != nil {
		return fmt.Errorf("failed to UpdateChannel: %w", err)
	}

	if args.Name.Valid || args.Parent.Valid {
		m.T.move(id, args.Parent, args.Name)
	}
	m.T.updateSingle(id, ch)

	updated := time.Now()
	for eventType, detail := range eventRecords {
		m.recordChannelEvent(id, eventType, detail, updated)
	}
	return nil
}

func (m *managerImpl) ArchiveChannel(id uuid.UUID, updaterID uuid.UUID) error {
	ch, err := m.GetChannel(id)
	if err != nil {
		return ErrChannelNotFound
	}

	if ch.IsArchived() {
		return nil // 既にアーカイブされている
	}
	if ch.IsDMChannel() {
		return ErrInvalidChannel // DMチャンネルはアーカイブ不可
	}

	m.T.Lock()
	defer m.T.Unlock()

	var (
		targets = []uuid.UUID{id}
		queue   = m.T.getChildrenIDs(id)
	)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if m.T.isArchivedChannel(id) {
			continue
		}

		targets = append(targets, id)
		queue = append(queue, m.T.getChildrenIDs(id)...)
	}

	chs, err := m.R.ArchiveChannels(targets)
	if err != nil {
		return fmt.Errorf("failed to ArchiveChannels: %w", err)
	}

	m.T.updateMultiple(chs)

	updated := time.Now()
	for _, ch := range chs {
		m.recordChannelEvent(ch.ID, model.ChannelEventVisibilityChanged, model.ChannelEventDetail{
			"userId":     updaterID,
			"visibility": ch.IsVisible,
		}, updated)
	}
	return nil
}

func (m *managerImpl) UnarchiveChannel(id uuid.UUID, updaterID uuid.UUID) error {
	ch, err := m.GetChannel(id)
	if err != nil {
		return ErrChannelNotFound
	}
	if !ch.IsArchived() {
		return nil // アーカイブされていない
	}

	m.T.Lock()
	defer m.T.Unlock()

	if m.T.isArchivedChannel(ch.ParentID) {
		return ErrInvalidParentChannel // 親チャンネルがアーカイブされている
	}

	ch, err = m.R.UpdateChannel(id, repository.UpdateChannelArgs{Visibility: optional.From(true)})
	if err != nil {
		return fmt.Errorf("failed to UpdateChannel: %w", err)
	}

	m.T.updateSingle(id, ch)

	m.recordChannelEvent(ch.ID, model.ChannelEventVisibilityChanged, model.ChannelEventDetail{
		"userId":     updaterID,
		"visibility": ch.IsVisible,
	}, ch.UpdatedAt)
	return nil
}

func (m *managerImpl) PublicChannelTree() Tree {
	return m.T
}

func (m *managerImpl) ChangeChannelSubscriptions(channelID uuid.UUID, subscriptions map[uuid.UUID]model.ChannelSubscribeLevel, keepOffLevel bool, updaterID uuid.UUID) error {
	if !m.IsPublicChannel(channelID) {
		return ErrInvalidChannel
	}
	if m.PublicChannelTree().IsForceChannel(channelID) {
		return ErrForcedNotification
	}

	on, off, err := m.R.ChangeChannelSubscription(channelID, repository.ChangeChannelSubscriptionArgs{
		Subscription: subscriptions,
		KeepOffLevel: keepOffLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to ChangeChannelSubscription: %w", err)
	}
	if len(on) > 0 || len(off) > 0 {
		m.recordChannelEvent(channelID, model.ChannelEventSubscribersChanged, model.ChannelEventDetail{
			"userId": updaterID,
			"on":     on,
			"off":    off,
		}, time.Now())
	}
	return nil
}

func (m *managerImpl) GetDMChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	if user1 == uuid.Nil || user2 == uuid.Nil {
		return nil, ErrChannelNotFound
	}

	ch, err := m.R.GetDirectMessageChannel(user1, user2)
	if err == nil {
		return ch, nil
	} else if err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to GetDirectMessageChannel: %w", err)
	}

	// 存在しなかったので作成
	ch, err = m.R.CreateChannel(
		model.Channel{
			Name:      "dm_" + random.AlphaNumeric(17),
			IsVisible: true,
		},
		set.UUIDSetFromArray([]uuid.UUID{user1, user2}),
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to CreateChannel: %w", err)
	}
	ch.ChildrenID = make([]uuid.UUID, 0)
	return ch, nil
}

func (m *managerImpl) GetDMChannelMembers(id uuid.UUID) ([]uuid.UUID, error) {
	members, err := m.R.GetPrivateChannelMemberIDs(id)
	if err != nil {
		return nil, fmt.Errorf("failed to GetDMCHannelMembers: %w", err)
	}
	return members, nil
}

func (m *managerImpl) GetDMChannelMapping(userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	mappings, err := m.R.GetDirectMessageChannelMapping(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to GetDMChannelMapping: %w", err)
	}

	result := map[uuid.UUID]uuid.UUID{}
	for _, ch := range mappings {
		if ch.User1 != userID {
			result[ch.ChannelID] = ch.User1
		} else {
			result[ch.ChannelID] = ch.User2
		}
	}
	return result, nil
}

func (m *managerImpl) IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
	if m.T.IsChannelPresent(channelID) {
		return true, nil // 公開チャンネルは全員アクセス可能
	}

	// DMチャンネル
	members, err := m.R.GetPrivateChannelMemberIDs(channelID)
	if err != nil {
		return false, fmt.Errorf("failed to IsChannelAccessibleToUser: %w", err)
	}
	for _, id := range members {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m *managerImpl) IsPublicChannel(id uuid.UUID) bool {
	return m.T.IsChannelPresent(id)
}

func (m *managerImpl) Wait() {
	m.P.Wait()
}

func (m *managerImpl) recordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) {
	m.P.Add(1)
	go func() {
		defer m.P.Done()

		err := m.R.RecordChannelEvent(channelID, eventType, detail, datetime)
		if err != nil {
			m.L.Warn("failed to record channel event", zap.Error(err), zap.Stringer("channelID", channelID), zap.Stringer("type", eventType), zap.Any("detail", detail), zap.Time("datetime", datetime))
		}
	}()
}
