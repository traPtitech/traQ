package bot

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/bot/event"
	"github.com/traPtitech/traQ/bot/payload"
	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
)

type eventHandler func(p *Processor, event string, fields hub.Fields)

var eventHandlerSet = map[string]eventHandler{
	intevent.BotJoined:           botJoinedOrLeftHandler,
	intevent.BotLeft:             botJoinedOrLeftHandler,
	intevent.BotPingRequest:      botPingRequestHandler,
	intevent.MessageCreated:      messageCreatedHandler,
	intevent.UserCreated:         userCreatedHandler,
	intevent.ChannelCreated:      channelCreatedHandler,
	intevent.ChannelTopicUpdated: channelTopicUpdatedHandler,
	intevent.StampCreated:        stampCreatedHandler,
	intevent.UserTagAdded:        userTagAddedHandler,
	intevent.UserTagRemoved:      userTagRemovedHandler,
}

func messageCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	m := fields["message"].(*model.Message)
	parsed := fields["parse_result"].(*message.ParseResult)

	ch, err := p.repo.GetChannel(m.ChannelID)
	if err != nil {
		p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", m.ChannelID))
		return
	}

	user, err := p.repo.GetUser(m.UserID, false)
	if err != nil {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", m.UserID))
		return
	}

	embedded, _ := message.ExtractEmbedding(m.Text)
	if ch.IsDMChannel() {
		ids, err := p.repo.GetPrivateChannelMemberIDs(ch.ID)
		if err != nil {
			p.logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("id", ch.ID))
			return
		}

		var id uuid.UUID
		for _, v := range ids {
			if v != m.UserID {
				id = v
				break
			}
		}

		bot, err := p.repo.GetBotByBotUserID(id)
		if err != nil {
			if err != repository.ErrNotFound {
				p.logger.Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("id", id))
			}
			return
		}
		if !filterBot(p, bot, stateFilter(model.BotActive), eventFilter(event.DirectMessageCreated), botUserIDNotEqualsFilter(m.UserID)) {
			return
		}

		p.unicast(
			event.DirectMessageCreated,
			payload.MakeDirectMessageCreated(m, user, embedded, parsed),
			bot,
		)
	} else {
		// 購読BOT
		query := repository.BotsQuery{}
		bots, err := p.repo.GetBots(query.CMemberOf(m.ChannelID).Active().Subscribe(event.MessageCreated))
		if err != nil {
			p.logger.Error("failed to GetBots", zap.Error(err))
			return
		}

		// メンションBOT
		done := make(map[uuid.UUID]bool)
		for _, uid := range parsed.Mentions {
			if !done[uid] {
				done[uid] = true
				b, err := p.repo.GetBotByBotUserID(uid)
				if err != nil {
					if err != repository.ErrNotFound {
						p.logger.Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("uid", uid))
					}
					continue
				}
				if b.SubscribeEvents.Contains(event.MentionMessageCreated) {
					bots = append(bots, b)
				}
			}
		}

		bots = filterBots(p, bots, stateFilter(model.BotActive), botUserIDNotEqualsFilter(m.UserID))
		if len(bots) == 0 {
			return
		}

		p.multicast(
			event.MessageCreated,
			payload.MakeMessageCreated(m, user, embedded, parsed),
			bots,
		)
	}
}

func botJoinedOrLeftHandler(p *Processor, ev string, fields hub.Fields) {
	botID := fields["bot_id"].(uuid.UUID)
	channelID := fields["channel_id"].(uuid.UUID)

	bot, err := p.repo.GetBotByID(botID)
	if err != nil {
		p.logger.Error("failed to GetBotByID", zap.Error(err), zap.Stringer("id", botID))
		return
	}
	if bot.State != model.BotActive {
		return
	}

	ch, err := p.repo.GetChannel(channelID)
	if err != nil {
		p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", channelID))
		return
	}
	user, err := p.repo.GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	switch ev {
	case intevent.BotJoined:
		p.unicast(
			event.Joined,
			payload.MakeJoinedOrLeft(ch, p.repo.GetChannelTree().GetChannelPath(channelID), user),
			bot,
		)
	case intevent.BotLeft:
		p.unicast(
			event.Left,
			payload.MakeJoinedOrLeft(ch, p.repo.GetChannelTree().GetChannelPath(channelID), user),
			bot,
		)
	}
}

func userCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	user := fields["user"].(model.UserInfo)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.Privileged().Active().Subscribe(event.UserCreated))
	if err != nil {
		p.logger.Error("failed to GetBots", zap.Error(err))
		return
	}

	p.multicast(
		event.UserCreated,
		payload.MakeUserCreated(user),
		bots,
	)
}

func channelCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	ch := fields["channel"].(*model.Channel)
	if ch.IsPublic {
		bots, err := p.repo.GetBots(repository.BotsQuery{}.Privileged().Active().Subscribe(event.ChannelCreated))
		if err != nil {
			p.logger.Error("failed to GetBots", zap.Error(err))
			return
		}
		if len(bots) == 0 {
			return
		}

		user, err := p.repo.GetUser(ch.CreatorID, false)
		if err != nil {
			p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
			return
		}

		p.multicast(
			event.ChannelCreated,
			payload.MakeChannelCreated(ch, p.repo.GetChannelTree().GetChannelPath(ch.ID), user),
			bots,
		)
	}
}

func channelTopicUpdatedHandler(p *Processor, _ string, fields hub.Fields) {
	chID := fields["channel_id"].(uuid.UUID)
	topic := fields["topic"].(string)
	updaterID := fields["updater_id"].(uuid.UUID)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.CMemberOf(chID).Active().Subscribe(event.ChannelTopicChanged))
	if err != nil {
		p.logger.Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	ch, err := p.repo.GetChannel(chID)
	if err != nil {
		p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", chID))
		return
	}

	chCreator, err := p.repo.GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	user, err := p.repo.GetUser(updaterID, false)
	if err != nil {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", updaterID))
		return
	}

	p.multicast(
		event.ChannelTopicChanged,
		payload.MakeChannelTopicChanged(ch, p.repo.GetChannelTree().GetChannelPath(ch.ID), chCreator, topic, user),
		bots,
	)
}

func stampCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	stamp := fields["stamp"].(*model.Stamp)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.Active().Subscribe(event.StampCreated))
	if err != nil {
		p.logger.Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	var user model.UserInfo
	if !stamp.IsSystemStamp() {
		user, err = p.repo.GetUser(stamp.CreatorID, false)
		if err != nil {
			p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", stamp.CreatorID))
			return
		}
	}

	p.multicast(
		event.StampCreated,
		payload.MakeStampCreated(stamp, user),
		bots,
	)
}

func botPingRequestHandler(p *Processor, _ string, fields hub.Fields) {
	bot := fields["bot"].(*model.Bot)

	buf, release, err := p.makePayloadJSON(payload.MakePing())
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	if p.sendEvent(bot, event.Ping, buf) {
		// OK
		if err := p.repo.ChangeBotState(bot.ID, model.BotActive); err != nil {
			p.logger.Error("failed to ChangeBotState", zap.Error(err))
		}
	} else {
		// NG
		if err := p.repo.ChangeBotState(bot.ID, model.BotPaused); err != nil {
			p.logger.Error("failed to ChangeBotState", zap.Error(err))
		}
	}
}

func userTagAddedHandler(p *Processor, _ string, fields hub.Fields) {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bot, err := p.repo.GetBotByBotUserID(userID)
	if err != nil {
		if err != repository.ErrNotFound {
			p.logger.Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("id", userID))
		}
		return
	}
	if bot.State != model.BotActive || !bot.SubscribeEvents.Contains(event.TagAdded) {
		return
	}

	t, err := p.repo.GetTagByID(tagID)
	if err != nil {
		p.logger.Error("failed to GetTagByID", zap.Error(err), zap.Stringer("id", tagID))
		return
	}

	p.unicast(
		event.TagAdded,
		payload.MakeTagAddedOrRemoved(t),
		bot,
	)
}

func userTagRemovedHandler(p *Processor, _ string, fields hub.Fields) {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bot, err := p.repo.GetBotByBotUserID(userID)
	if err != nil {
		if err != repository.ErrNotFound {
			p.logger.Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("id", userID))
		}
		return
	}
	if bot.State != model.BotActive || !bot.SubscribeEvents.Contains(event.TagRemoved) {
		return
	}

	t, err := p.repo.GetTagByID(tagID)
	if err != nil {
		p.logger.Error("failed to GetTagByID", zap.Error(err), zap.Stringer("id", tagID))
		return
	}

	p.unicast(
		event.TagRemoved,
		payload.MakeTagAddedOrRemoved(t),
		bot,
	)
}
