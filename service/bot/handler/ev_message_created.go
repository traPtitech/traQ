package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
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

	embedded, _ := message.ExtractEmbedding(m.Text)
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

		bots, err := ctx.R().GetBots(repository.BotsQuery{}.Active().Subscribe(event.DirectMessageCreated).BotUserID(id))
		if err != nil {
			ctx.L().Error("failed to GetBots", zap.Error(err))
			return
		}
		if len(bots) == 0 {
			return
		}

		if err := event.Unicast(
			ctx.D(),
			event.DirectMessageCreated,
			payload.MakeDirectMessageCreated(m, user, embedded, parsed),
			bots[0],
		); err != nil {
			ctx.L().Error("failed to unicast", zap.Error(err))
		}
	} else {
		// 購読BOT
		query := repository.BotsQuery{}
		bots, err := ctx.R().GetBots(query.CMemberOf(m.ChannelID).Active().Subscribe(event.MessageCreated))
		if err != nil {
			ctx.L().Error("failed to GetBots", zap.Error(err))
			return
		}

		// メンションBOT
		done := make(map[uuid.UUID]bool)
		for _, uid := range parsed.Mentions {
			if !done[uid] {
				done[uid] = true
				b, err := ctx.R().GetBotByBotUserID(uid)
				if err != nil {
					if err != repository.ErrNotFound {
						ctx.L().Error("failed to GetBotByBotUserID", zap.Error(err), zap.Stringer("uid", uid))
					}
					continue
				}
				if b.State != model.BotActive {
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

		if err := event.Multicast(
			ctx.D(),
			event.MessageCreated,
			payload.MakeMessageCreated(m, user, embedded, parsed),
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
