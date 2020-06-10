package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
)

func MessageCreated(ctx Context, _ string, fields hub.Fields) {
	m := fields["message"].(*model.Message)
	parsed := fields["parse_result"].(*message.ParseResult)

	ch, err := ctx.CM().GetChannel(m.ChannelID)
	if err != nil {
		ctx.L().Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", m.ChannelID))
		return
	}

	user, err := ctx.R().GetUser(m.UserID, false)
	if err != nil {
		ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", m.UserID))
		return
	}

	if ch.IsDMChannel() {
		ids, err := ctx.CM().GetDMChannelMembers(ch.ID)
		if err != nil {
			ctx.L().Error("failed to GetDMChannelMembers", zap.Error(err), zap.Stringer("id", ch.ID))
			return
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
			ctx.L().Error("failed to GetBotByBotUserID", zap.Error(err))
			return
		}
		if bot == nil || !bot.SubscribeEvents.Contains(event.DirectMessageCreated) {
			return
		}

		if err := ctx.Unicast(
			event.DirectMessageCreated,
			payload.MakeDirectMessageCreated(m, user, parsed),
			bot,
		); err != nil {
			ctx.L().Error("failed to unicast", zap.Error(err))
		}
	} else {
		// 購読BOT
		bots, err := ctx.GetChannelBots(m.ChannelID, event.MessageCreated)
		if err != nil {
			ctx.L().Error("failed to GetChannelBots", zap.Error(err))
			return
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
			return
		}

		if err := ctx.Multicast(
			event.MessageCreated,
			payload.MakeMessageCreated(m, user, parsed),
			bots,
		); err != nil {
			ctx.L().Error("failed to multicast", zap.Error(err))
		}
	}
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
