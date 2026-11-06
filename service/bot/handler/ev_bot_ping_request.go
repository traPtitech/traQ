package handler

import (
	"context"
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/utils/jsonutil"
)

func BotPingRequest(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	bot := fields["bot"].(*model.Bot)

	buf, err := jsonutil.Marshal(payload.MakePing(datetime))
	if err != nil {
		return err
	}

	if ctx.D().Send(bot, event.Ping, buf) {
		// OK
		if err := ctx.R().ChangeBotState(context.Background(), bot.ID, model.BotActive); err != nil {
			return fmt.Errorf("failed to ChangeBotState: %w", err)
		}
	} else {
		// NG
		if err := ctx.R().ChangeBotState(context.Background(), bot.ID, model.BotPaused); err != nil {
			return fmt.Errorf("failed to ChangeBotState: %w", err)
		}
	}
	return nil
}
