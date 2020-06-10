package handler

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func BotPingRequest(ctx Context, _ string, fields hub.Fields) {
	bot := fields["bot"].(*model.Bot)

	buf, err := jsoniter.ConfigFastest.Marshal(payload.MakePing())
	if err != nil {
		ctx.L().Error("unexpected json encode error", zap.Error(err))
		return
	}

	if ctx.D().Send(bot, event.Ping, buf) {
		// OK
		if err := ctx.R().ChangeBotState(bot.ID, model.BotActive); err != nil {
			ctx.L().Error("failed to ChangeBotState", zap.Error(err))
		}
	} else {
		// NG
		if err := ctx.R().ChangeBotState(bot.ID, model.BotPaused); err != nil {
			ctx.L().Error("failed to ChangeBotState", zap.Error(err))
		}
	}
}
