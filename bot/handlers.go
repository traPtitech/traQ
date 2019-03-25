package bot

import (
	"bytes"
	"encoding/json"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	"time"
)

func (p *Processor) pingHandler(bot *model.Bot) {
	payload := pingPayload{
		basePayload: basePayload{
			EventTime: time.Now(),
		},
	}

	buf := p.bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		p.bufPool.Put(buf)
	}()

	if err := json.NewEncoder(buf).Encode(&payload); err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}

	if p.sendEvent(bot, Ping, buf.Bytes()) {
		// OK
		if err := p.repo.ChangeBotStatus(bot.ID, model.BotActive); err != nil {
			p.logger.Error("failed to ChangeBotStatus", zap.Error(err))
		}
	} else {
		// NG
		if err := p.repo.ChangeBotStatus(bot.ID, model.BotPaused); err != nil {
			p.logger.Error("failed to ChangeBotStatus", zap.Error(err))
		}
	}
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
		basePayload: basePayload{
			EventTime: time.Now(),
		},
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

	buf := p.bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		p.bufPool.Put(buf)
	}()

	if err := json.NewEncoder(buf).Encode(&payload); err != nil {
		p.logger.Error("unexpected json encode error", zap.Error(err))
		return
	}
	for _, bot := range bots {
		if message.UserID == bot.BotUserID {
			continue // Bot自身の発言はスキップ
		}
		p.sendEvent(bot, MessageCreated, buf.Bytes())
	}
}

func filterBots(bots []*model.Bot, event model.BotEvent) []*model.Bot {
	result := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		if bot.Status != model.BotActive {
			continue
		}
		if !bot.SubscribeEvents.Contains(event) {
			continue
		}
		result = append(result, bot)
	}
	return result
}
