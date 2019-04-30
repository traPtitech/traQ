package bot

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
)

type eventHandler func(p *Processor, event string, fields hub.Fields)

var eventHandlerSet = map[string]eventHandler{
	event.BotJoined:      botJoinedAndLeftHandler,
	event.BotLeft:        botJoinedAndLeftHandler,
	event.BotPingRequest: botPingRequestHandler,
	event.MessageCreated: messageCreatedHandler,
	event.UserCreated:    userCreatedHandler,
	event.ChannelCreated: channelCreatedHandler,
}

func messageCreatedHandler(p *Processor, _ string, fields hub.Fields) {
	m := fields["message"].(*model.Message)
	embedded := fields["embedded"].([]*message.EmbeddedInfo)
	plain := fields["plain"].(string)

	bots, err := p.repo.GetBotsByChannel(m.ChannelID)
	if err != nil {
		p.logger.Error("failed to GetBotsByChannel", zap.Error(err))
		return
	}
	bots = filterBots(p, bots, stateFilter(model.BotActive), eventFilter(MessageCreated), botUserIDNotEqualsFilter(m.UserID))
	if len(bots) == 0 {
		return
	}

	payload := messageCreatedPayload{
		basePayload: makeBasePayload(),
		Message:     makeMessagePayload(m, embedded, plain),
	}

	multicast(p, MessageCreated, &payload, bots)
}

func botJoinedAndLeftHandler(p *Processor, ev string, fields hub.Fields) {
	botID := fields["bot_id"].(uuid.UUID)
	channelID := fields["channel_id"].(uuid.UUID)

	bot, err := p.repo.GetBotByID(botID)
	if err != nil {
		p.logger.Error("failed to GetBotByID", zap.Error(err), zap.Stringer("id", botID))
		return
	}

	if filterBot(p, bot, stateFilter(model.BotActive)) {
		return
	}

	payload := joinAndLeftPayload{
		basePayload: makeBasePayload(),
		ChannelID:   channelID,
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

		payload := channelCreatedPayload{
			basePayload: makeBasePayload(),
			Channel: channelPayload{
				ID:        chID,
				Name:      ch.Name,
				Path:      "#" + path,
				ParentID:  ch.ParentID,
				CreatorID: ch.CreatorID,
				CreatedAt: ch.CreatedAt,
				UpdatedAt: ch.UpdatedAt,
			},
		}

		multicast(p, ChannelCreated, &payload, bots)
	}
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

	for _, bot := range targets {
		p.sendEvent(bot, ev, buf)
	}
}
