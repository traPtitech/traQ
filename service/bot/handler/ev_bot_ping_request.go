package handler

import (
	"fmt"
	"time"

	jsonIter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func BotPingRequest(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	bot := fields["bot"].(*model.Bot)

	buf, err := jsonIter.ConfigFastest.Marshal(payload.MakePing(datetime))
	if err != nil {
		return err
	}

	if ctx.D().Send(bot, event.Ping, buf) {
		// OK
		if err := ctx.R().ChangeBotState(bot.ID, model.BotActive); err != nil {
			return fmt.Errorf("failed to ChangeBotState: %w", err)
		}
	} else {
		// NG
		if err := ctx.R().ChangeBotState(bot.ID, model.BotPaused); err != nil {
			return fmt.Errorf("failed to ChangeBotState: %w", err)
		}
	}
	return nil
}
