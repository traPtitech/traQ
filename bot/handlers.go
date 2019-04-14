package bot

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
)

func (p *Processor) pingHandler(bot *model.Bot) {
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

func (p *Processor) joinedAndLeftHandler(botId, channelId uuid.UUID, ev model.BotEvent) {
	bot, err := p.repo.GetBotByID(botId)
	if err != nil {
		p.logger.Error("failed to GetBotByID", zap.Error(err), zap.Stringer("id", botId))
		return
	}
	if bot.State != model.BotActive {
		return
	}

	payload := joinAndLeftPayload{
		basePayload: makeBasePayload(),
		ChannelId:   channelId,
	}

	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	p.sendEvent(bot, ev, buf)
}

func (p *Processor) createMessageHandler(message *model.Message, embedded []*message.EmbeddedInfo, plain string) {
	bots, err := p.repo.GetBotsByChannel(message.ChannelID)
	if err != nil {
		p.logger.Error("failed to GetBotsByChannel", zap.Error(err))
		return
	}
	bots = filterBots(bots, MessageCreated)
	if len(bots) == 0 {
		return
	}

	payload := messageCreatedPayload{
		basePayload: makeBasePayload(),
		Message: messagePayload{
			ID:        message.ID,
			UserID:    message.UserID,
			ChannelID: message.ChannelID,
			Text:      message.Text,
			PlainText: plain,
			Embedded:  embedded,
			CreatedAt: message.CreatedAt,
			UpdatedAt: message.UpdatedAt,
		},
	}

	buf, release, err := p.makePayloadJSON(&payload)
	if err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	defer release()

	for _, bot := range bots {
		if message.UserID == bot.BotUserID {
			continue // Bot自身の発言はスキップ
		}
		p.sendEvent(bot, MessageCreated, buf)
	}
}

func filterBots(bots []*model.Bot, event model.BotEvent) []*model.Bot {
	result := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		if bot.State != model.BotActive {
			continue
		}
		if !bot.SubscribeEvents.Contains(event) {
			continue
		}
		result = append(result, bot)
	}
	return result
}
