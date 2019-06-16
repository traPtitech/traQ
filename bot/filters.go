package bot

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

type filterFunc func(p *Processor, bot *model.Bot) bool

func filterBots(p *Processor, bots []*model.Bot, filters ...filterFunc) []*model.Bot {
	result := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		if filterBot(p, bot, filters...) {
			result = append(result, bot)
		}
	}
	return result
}

func filterBot(p *Processor, bot *model.Bot, filters ...filterFunc) bool {
	for _, v := range filters {
		if !v(p, bot) {
			return false
		}
	}
	return true
}

func stateFilter(state model.BotState) filterFunc {
	return func(p *Processor, bot *model.Bot) bool {
		return bot.State == state
	}
}

func privilegedFilter() filterFunc {
	return func(p *Processor, bot *model.Bot) bool {
		return bot.Privileged
	}
}

func eventFilter(event model.BotEvent) filterFunc {
	return func(p *Processor, bot *model.Bot) bool {
		return bot.SubscribeEvents.Contains(event)
	}
}

func botUserIDNotEqualsFilter(id uuid.UUID) filterFunc {
	return func(p *Processor, bot *model.Bot) bool {
		return id != bot.BotUserID
	}
}
