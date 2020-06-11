package handler

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	"time"
)

func MessageCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	m := fields["message"].(*model.Message)
	parsed := fields["parse_result"].(*message.ParseResult)

	ch, err := ctx.CM().GetChannel(m.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	user, err := ctx.R().GetUser(m.UserID, false)
	if err != nil {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	if ch.IsDMChannel() {
		ids, err := ctx.CM().GetDMChannelMembers(ch.ID)
		if err != nil {
			return fmt.Errorf("failed to GetDMChannelMembers: %w", err)
		}

		var id uuid.UUID
		for _, v := range ids {
			if v != m.UserID {
				id = v
				break
			}
		}

		bot, err := ctx.GetBotByBotUserID(id)
		if err != nil {
			return fmt.Errorf("failed to GetBotByBotUserID: %w", err)
		}
		if bot == nil || !bot.SubscribeEvents.Contains(event.DirectMessageCreated) {
			return nil
		}

		if err := ctx.Unicast(
			event.DirectMessageCreated,
			payload.MakeDirectMessageCreated(datetime, m, user, parsed),
			bot,
		); err != nil {
			return fmt.Errorf("failed to unicast: %w", err)
		}
	} else {
		// 購読BOT
		bots, err := ctx.GetChannelBots(m.ChannelID, event.MessageCreated)
		if err != nil {
			return fmt.Errorf("failed to GetChannelBots: %w", err)
		}

		// メンションBOT
		done := make(map[uuid.UUID]bool)
		for _, uid := range parsed.Mentions {
			if !done[uid] {
				done[uid] = true
				b, err := ctx.GetBotByBotUserID(uid)
				if err != nil {
					ctx.L().Error("failed to GetBotByBotUserID", zap.Error(err))
					continue
				}
				if b == nil {
					continue
				}
				if b.SubscribeEvents.Contains(event.MentionMessageCreated) {
					bots = append(bots, b)
				}
			}
		}

		bots = filterBotUserIDNotEquals(bots, m.UserID)
		if len(bots) == 0 {
			return nil
		}

		if err := ctx.Multicast(
			event.MessageCreated,
			payload.MakeMessageCreated(datetime, m, user, parsed),
			bots,
		); err != nil {
			return fmt.Errorf("failed to multicast: %w", err)
		}
	}
	return nil
}

func filterBotUserIDNotEquals(bots []*model.Bot, id uuid.UUID) []*model.Bot {
	result := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		if bot.BotUserID != id {
			result = append(result, bot)
		}
	}
	return result
}
