package bot

import (
	"sync"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
)

type eventHandler func(p *Processor, event string, fields hub.Fields)

var eventHandlerSet = map[string]eventHandler{
	event.BotJoined:           botJoinedAndLeftHandler,
	event.BotLeft:             botJoinedAndLeftHandler,
	event.BotPingRequest:      botPingRequestHandler,
	event.MessageCreated:      messageCreatedHandler,
	event.UserCreated:         userCreatedHandler,
	event.ChannelCreated:      channelCreatedHandler,
	event.ChannelTopicUpdated: channelTopicUpdatedHandler,
	event.StampCreated:        stampCreatedHandler,
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
		if !filterBot(p, bot, stateFilter(model.BotActive), eventFilter(model.BotEventDirectMessageCreated), botUserIDNotEqualsFilter(m.UserID)) {
			return
		}

		payload := directMessageCreatedPayload{
			basePayload: makeBasePayload(),
			Message:     makeMessagePayload(m, user, embedded, parsed.PlainText),
		}

		multicast(p, model.BotEventDirectMessageCreated, &payload, []*model.Bot{bot})
	} else {
		// 購読BOT
		query := repository.BotsQuery{}
		bots, err := p.repo.GetBots(query.CMemberOf(m.ChannelID).Active().Subscribe(model.BotEventMessageCreated))
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
				if b.SubscribeEvents.Contains(model.BotEventMentionMessageCreated) {
					bots = append(bots, b)
				}
			}
		}

		bots = filterBots(p, bots, stateFilter(model.BotActive), botUserIDNotEqualsFilter(m.UserID))
		if len(bots) == 0 {
			return
		}

		payload := messageCreatedPayload{
			basePayload: makeBasePayload(),
			Message:     makeMessagePayload(m, user, embedded, parsed.PlainText),
		}

		multicast(p, model.BotEventMessageCreated, &payload, bots)
	}
}

func botJoinedAndLeftHandler(p *Processor, ev string, fields hub.Fields) {
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
	path, err := p.repo.GetChannelPath(channelID)
	if err != nil {
		p.logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("id", channelID))
		return
	}
	user, err := p.repo.GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	payload := joinAndLeftPayload{
		basePayload: makeBasePayload(),
		Channel:     makeChannelPayload(ch, path, user),
	}

	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	switch ev {
	case event.BotJoined:
		p.sendEvent(bot, model.BotEventJoined, buf)
	case event.BotLeft:
		p.sendEvent(bot, model.BotEventLeft, buf)
	}
}

func userCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	user := fields["user"].(model.UserInfo)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.Privileged().Active().Subscribe(model.BotEventUserCreated))
	if err != nil {
		p.logger.Error("failed to GetBots", zap.Error(err))
		return
	}

	multicast(p, model.BotEventUserCreated, &userCreatedPayload{
		basePayload: makeBasePayload(),
		User:        makeUserPayload(user),
	}, bots)
}

func channelCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	ch := fields["channel"].(*model.Channel)
	private := fields["private"].(bool)

	if !private {
		bots, err := p.repo.GetBots(repository.BotsQuery{}.Privileged().Active().Subscribe(model.BotEventChannelCreated))
		if err != nil {
			p.logger.Error("failed to GetBots", zap.Error(err))
			return
		}
		if len(bots) == 0 {
			return
		}

		path, err := p.repo.GetChannelPath(ch.ID)
		if err != nil {
			p.logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("id", ch.ID))
			return
		}
		user, err := p.repo.GetUser(ch.CreatorID, false)
		if err != nil {
			p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
			return
		}

		multicast(p, model.BotEventChannelCreated, &channelCreatedPayload{
			basePayload: makeBasePayload(),
			Channel:     makeChannelPayload(ch, path, user),
		}, bots)
	}
}

func channelTopicUpdatedHandler(p *Processor, _ string, fields hub.Fields) {
	chID := fields["channel_id"].(uuid.UUID)
	topic := fields["topic"].(string)
	updaterID := fields["updater_id"].(uuid.UUID)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.CMemberOf(chID).Active().Subscribe(model.BotEventChannelTopicChanged))
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

	path, err := p.repo.GetChannelPath(ch.ID)
	if err != nil {
		p.logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("id", ch.ID))
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

	multicast(p, model.BotEventChannelTopicChanged, &channelTopicChangedPayload{
		basePayload: makeBasePayload(),
		Channel:     makeChannelPayload(ch, path, chCreator),
		Topic:       topic,
		Updater:     makeUserPayload(user),
	}, bots)
}

func stampCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	stamp := fields["stamp"].(*model.Stamp)

	bots, err := p.repo.GetBots(repository.BotsQuery{}.Active().Subscribe(model.BotEventStampCreated))
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

	multicast(p, model.BotEventStampCreated, &stampCreatedPayload{
		basePayload: makeBasePayload(),
		ID:          stamp.ID,
		Name:        stamp.Name,
		FileID:      stamp.FileID,
		Creator:     makeUserPayload(user),
	}, bots)
}

func botPingRequestHandler(p *Processor, _ string, fields hub.Fields) {
	bot := fields["bot"].(*model.Bot)

	buf, release, err := p.makePayloadJSON(&pingPayload{basePayload: makeBasePayload()})
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	if p.sendEvent(bot, model.BotEventPing, buf) {
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

func multicast(p *Processor, ev model.BotEvent, payload interface{}, targets []*model.Bot) {
	if len(targets) == 0 {
		return
	}
	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	var wg sync.WaitGroup
	done := make(map[uuid.UUID]bool, len(targets))
	for _, bot := range targets {
		if !done[bot.ID] {
			done[bot.ID] = true
			bot := bot
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.sendEvent(bot, ev, buf)
			}()
		}
	}
	wg.Wait()
}
