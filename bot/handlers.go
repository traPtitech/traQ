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
	embedded := fields["embedded"].([]*message.EmbeddedInfo)
	plain := fields["plain"].(string)

	ch, err := p.repo.GetChannel(m.ChannelID)
	if err != nil {
		p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", m.ChannelID))
		return
	}

	user, err := p.repo.GetUser(m.UserID)
	if err != nil {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", m.UserID))
		return
	}

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
		if !filterBot(p, bot, stateFilter(model.BotActive), eventFilter(DirectMessageCreated), botUserIDNotEqualsFilter(m.UserID)) {
			return
		}

		payload := directMessageCreatedPayload{
			basePayload: makeBasePayload(),
			Message:     makeMessagePayload(m, user, embedded, plain),
		}

		multicast(p, DirectMessageCreated, &payload, []*model.Bot{bot})
	} else {
		// 購読BOT
		bots, err := p.repo.GetBotsByChannel(m.ChannelID)
		if err != nil {
			p.logger.Error("failed to GetBotsByChannel", zap.Error(err), zap.Stringer("id", m.ChannelID))
			return
		}

		// メンションBOT
		for _, v := range embedded {
			if v.Type == "user" {
				uid, err := uuid.FromString(v.ID)
				if err != nil {
					b, err := p.repo.GetBotByBotUserID(uid)
					if err != nil {
						if err != repository.ErrNotFound {
							p.logger.Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("uid", uid))
						}
						continue
					}
					bots = append(bots, b)
				}
			}
		}

		bots = filterBots(p, bots, stateFilter(model.BotActive), eventFilter(MessageCreated), botUserIDNotEqualsFilter(m.UserID))
		if len(bots) == 0 {
			return
		}

		payload := messageCreatedPayload{
			basePayload: makeBasePayload(),
			Message:     makeMessagePayload(m, user, embedded, plain),
		}

		multicast(p, MessageCreated, &payload, bots)
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

	if !filterBot(p, bot, stateFilter(model.BotActive)) {
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
	user, err := p.repo.GetUser(ch.CreatorID)
	if err != nil {
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
		p.sendEvent(bot, Joined, buf)
	case event.BotLeft:
		p.sendEvent(bot, Left, buf)
	}
}

func userCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	user := fields["user"].(*model.User)

	bots, err := p.repo.GetAllBots()
	if err != nil {
		p.logger.Error("failed to GetAllBots", zap.Error(err))
		return
	}
	bots = filterBots(p, bots, privilegedFilter(), stateFilter(model.BotActive), eventFilter(UserCreated))
	if len(bots) == 0 {
		return
	}

	payload := userCreatedPayload{
		basePayload: makeBasePayload(),
		User:        makeUserPayload(user),
	}

	multicast(p, UserCreated, &payload, bots)
}

func channelCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	chID := fields["channel_id"].(uuid.UUID)
	private := fields["private"].(bool)

	bots, err := p.repo.GetAllBots()
	if err != nil {
		p.logger.Error("failed to GetAllBots", zap.Error(err))
		return
	}
	if !private {
		bots = filterBots(p, bots, privilegedFilter(), stateFilter(model.BotActive), eventFilter(ChannelCreated))
		if len(bots) == 0 {
			return
		}

		ch, err := p.repo.GetChannel(chID)
		if err != nil {
			p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", chID))
			return
		}
		path, err := p.repo.GetChannelPath(chID)
		if err != nil {
			p.logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("id", chID))
			return
		}
		user, err := p.repo.GetUser(ch.CreatorID)
		if err != nil {
			p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
			return
		}

		payload := channelCreatedPayload{
			basePayload: makeBasePayload(),
			Channel:     makeChannelPayload(ch, path, user),
		}

		multicast(p, ChannelCreated, &payload, bots)
	}
}

func channelTopicUpdatedHandler(p *Processor, _ string, fields hub.Fields) {
	chID := fields["channel_id"].(uuid.UUID)
	topic := fields["topic"].(string)
	updaterID := fields["updater_id"].(uuid.UUID)

	bots, err := p.repo.GetBotsByChannel(chID)
	if err != nil {
		p.logger.Error("failed to GetBotsByChannel", zap.Error(err), zap.Stringer("id", chID))
		return
	}
	bots = filterBots(p, bots, stateFilter(model.BotActive), eventFilter(ChannelTopicChanged))
	if len(bots) == 0 {
		return
	}

	ch, err := p.repo.GetChannel(chID)
	if err != nil {
		p.logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", chID))
		return
	}

	path, err := p.repo.GetChannelPath(chID)
	if err != nil {
		p.logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("id", chID))
		return
	}

	chCreator, err := p.repo.GetUser(ch.CreatorID)
	if err != nil {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	user, err := p.repo.GetUser(updaterID)
	if err != nil {
		p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", updaterID))
		return
	}

	payload := channelTopicChangedPayload{
		basePayload: makeBasePayload(),
		Channel:     makeChannelPayload(ch, path, chCreator),
		Topic:       topic,
		Updater:     makeUserPayload(user),
	}

	multicast(p, ChannelTopicChanged, &payload, bots)
}

func stampCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	stamp := fields["stamp"].(*model.Stamp)

	bots, err := p.repo.GetAllBots()
	if err != nil {
		p.logger.Error("failed to GetAllBots", zap.Error(err))
		return
	}
	bots = filterBots(p, bots, stateFilter(model.BotActive), eventFilter(StampCreated))
	if len(bots) == 0 {
		return
	}

	payload := stampCreatedPayload{
		basePayload: makeBasePayload(),
		ID:          stamp.ID,
		Name:        stamp.Name,
		FileID:      stamp.FileID,
	}

	if stamp.CreatorID != uuid.Nil {
		user, err := p.repo.GetUser(stamp.CreatorID)
		if err != nil {
			p.logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("id", stamp.CreatorID))
			return
		}
		payload.Creator = makeUserPayload(user)
	}

	multicast(p, StampCreated, &payload, bots)
}

func botPingRequestHandler(p *Processor, _ string, fields hub.Fields) {
	botID := fields["bot_id"].(uuid.UUID)
	bot, err := p.repo.GetBotByID(botID)
	if err != nil {
		p.logger.Error("failed to GetBotByID", zap.Error(err), zap.Stringer("bot_id", botID))
		return
	}

	payload := pingPayload{
		basePayload: makeBasePayload(),
	}

	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	if p.sendEvent(bot, Ping, buf) {
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
	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	var wg sync.WaitGroup
	for _, bot := range targets {
		bot := bot
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.sendEvent(bot, ev, buf)
		}()
	}
	wg.Wait()
}
