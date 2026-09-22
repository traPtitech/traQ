package handler

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"

	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/optional"
)

func TestMessageCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:        uuid.NewV3(uuid.Nil, "b"),
		BotUserID: uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{
			event.MessageCreated.String(),
			event.DirectMessageCreated.String(),
		}),
		State: model.BotActive,
	}
	bu := &model.User{
		ID:     b.BotUserID,
		Name:   "bot",
		Status: model.UserAccountStatusActive,
		Bot:    true,
	}
	ch := &model.Channel{
		ID:   uuid.NewV3(uuid.Nil, "c"),
		Name: "test",
		Type: model.ChannelTypePublic,
	}

	t.Run("success (public message, sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "u"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		parsed := message.Parse(m.Text)
		mu := &model.User{
			ID:   m.UserID,
			Name: "testman",
		}
		registerUser(repo, mu)
		registerChannel(cm, ch)
		et := time.Now()

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageCreated).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		expectMulticast(handlerCtx, event.MessageCreated, payload.MakeMessageCreated(et, m, mu, parsed), []*model.Bot{b})
		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})

	t.Run("success (public message, no targets)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "bu"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mu := &model.User{
			ID:   m.UserID,
			Name: "BOT_TEST",
		}
		registerUser(repo, mu)
		registerChannel(cm, ch)

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageCreated).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		assert.NoError(t, MessageCreated(handlerCtx, time.Now(), intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": message.Parse(m.Text),
		}))
	})

	t.Run("success (dm)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)
		dmc, u := createDMChannel(handlerCtx, cm, repo, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    u.GetID(),
			ChannelID: dmc.ID,
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		parsed := message.Parse(m.Text)
		et := time.Now()

		expectUnicast(handlerCtx, event.DirectMessageCreated, payload.MakeDirectMessageCreated(et, m, u, parsed), b)
		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})

	t.Run("success (dm, no sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)
		registerUser(repo, bu)
		dmc, _ := createDMChannel(handlerCtx, cm, repo, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    b.BotUserID,
			ChannelID: dmc.ID,
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		parsed := message.Parse(m.Text)
		et := time.Now()

		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})
}

func TestMessageCreatedMentions(t *testing.T) {
	t.Parallel()

	b1 := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b1"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu1"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.MentionMessageCreated.String()}),
		State:           model.BotActive,
	}
	b2 := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b2"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu2"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.MentionMessageCreated.String()}),
		State:           model.BotActive,
	}
	unsubscribed := &model.Bot{
		ID:        uuid.NewV3(uuid.Nil, "unsubscribed"),
		BotUserID: uuid.NewV3(uuid.Nil, "unsubscribed-user"),
		State:     model.BotActive,
	}
	defaultSenderID := uuid.NewV3(uuid.Nil, "u")
	failedBotUserID := uuid.NewV3(uuid.Nil, "failed-bot-user")
	groupID1 := uuid.NewV3(uuid.Nil, "g1")
	groupID2 := uuid.NewV3(uuid.Nil, "g2")
	botMention1 := fmt.Sprintf(`!{"type":"user","raw":"@bot","id":"%s"}`, b1.BotUserID)
	groupMention1 := fmt.Sprintf(`!{"type":"group","raw":"@group1","id":"%s"}`, groupID1)
	groupMention2 := fmt.Sprintf(`!{"type":"group","raw":"@group2","id":"%s"}`, groupID2)
	lookupErr := errors.New("lookup failed")

	for _, tt := range []struct {
		name        string
		text        string
		groups      map[uuid.UUID][]uuid.UUID
		groupErrors map[uuid.UUID]error
		senderID    uuid.UUID
		channelBots []*model.Bot
		wantBots    []*model.Bot
	}{
		{
			name:     "user mention",
			text:     botMention1,
			wantBots: []*model.Bot{b1},
		},
		{
			name:     "group mention sends only to subscribed bots",
			text:     groupMention1,
			groups:   map[uuid.UUID][]uuid.UUID{groupID1: {b1.BotUserID, unsubscribed.BotUserID, b2.BotUserID}},
			wantBots: []*model.Bot{b1, b2},
		},
		{
			name: "overlapping user and group mentions",
			text: botMention1 + groupMention1 + groupMention2 + groupMention1 + botMention1,
			groups: map[uuid.UUID][]uuid.UUID{
				groupID1: {b1.BotUserID, b2.BotUserID},
				groupID2: {b2.BotUserID, b1.BotUserID},
			},
			wantBots: []*model.Bot{b1, b2},
		},
		{
			name:   "group has no subscribed bots",
			text:   groupMention1,
			groups: map[uuid.UUID][]uuid.UUID{groupID1: {unsubscribed.BotUserID}},
		},
		{
			name:   "group has no bots or does not exist",
			text:   groupMention1,
			groups: map[uuid.UUID][]uuid.UUID{groupID1: {}},
		},
		{
			name:     "bot does not receive its own group mention",
			text:     groupMention1,
			groups:   map[uuid.UUID][]uuid.UUID{groupID1: {b1.BotUserID, b2.BotUserID}},
			senderID: b1.BotUserID,
			wantBots: []*model.Bot{b2},
		},
		{
			name:        "group lookup error preserves user mentions and other groups",
			text:        groupMention1 + botMention1 + groupMention2,
			groups:      map[uuid.UUID][]uuid.UUID{groupID2: {b2.BotUserID}},
			groupErrors: map[uuid.UUID]error{groupID1: lookupErr},
			wantBots:    []*model.Bot{b1, b2},
		},
		{
			name:        "group lookup error preserves channel subscriptions",
			text:        groupMention1,
			groupErrors: map[uuid.UUID]error{groupID1: lookupErr},
			channelBots: []*model.Bot{b2},
			wantBots:    []*model.Bot{b2},
		},
		{
			name:        "group lookup error with no other recipients",
			text:        groupMention1,
			groupErrors: map[uuid.UUID]error{groupID1: lookupErr},
		},
		{
			name:     "bot lookup error preserves other group members",
			text:     groupMention1,
			groups:   map[uuid.UUID][]uuid.UUID{groupID1: {failedBotUserID, b1.BotUserID}},
			wantBots: []*model.Bot{b1},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			handlerCtx, cm, repo := setup(t, ctrl)
			for _, b := range []*model.Bot{b1, b2, unsubscribed} {
				registerBot(t, handlerCtx, b)
			}
			handlerCtx.EXPECT().GetBotByBotUserID(failedBotUserID).Return(nil, lookupErr).AnyTimes()

			senderID := tt.senderID
			if senderID == uuid.Nil {
				senderID = defaultSenderID
			}
			ch := &model.Channel{ID: uuid.NewV3(uuid.Nil, "c"), IsPublic: true}
			mu := &model.User{ID: senderID, Name: "sender"}
			m := &model.Message{
				ID:        uuid.NewV3(uuid.Nil, "m"),
				UserID:    senderID,
				ChannelID: ch.ID,
				Text:      tt.text,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			parsed := message.Parse(m.Text)
			registerUser(repo, mu)
			registerChannel(cm, ch)
			handlerCtx.EXPECT().GetChannelBots(ch.ID, event.MessageCreated).Return(tt.channelBots, nil)
			for gid, members := range tt.groups {
				repo.MockUserRepository.EXPECT().
					GetUserIDs(gomock.Any(), repository.UsersQuery{IsBot: optional.From(true)}.GMemberOf(gid)).
					Return(members, nil)
			}
			for gid, err := range tt.groupErrors {
				repo.MockUserRepository.EXPECT().
					GetUserIDs(gomock.Any(), repository.UsersQuery{IsBot: optional.From(true)}.GMemberOf(gid)).
					Return(nil, err)
			}

			et := time.Now()
			if len(tt.wantBots) > 0 {
				handlerCtx.EXPECT().Multicast(event.MessageCreated, payload.MakeMessageCreated(et, m, mu, parsed), gomock.InAnyOrder(tt.wantBots))
			}
			assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
				"message_id":   m.ID,
				"message":      m,
				"parse_result": parsed,
			}))
			assert.Equal(t, message.Parse(m.Text), parsed)
		})
	}
}
