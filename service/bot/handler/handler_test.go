package handler

import (
	"github.com/golang/mock/gomock"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/bot/handler/mock_handler"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/testutils"
	"go.uber.org/zap"
	"testing"
)

type Repo struct {
	*mock_repository.MockTagRepository
	*mock_repository.MockUserRepository
	*mock_repository.MockBotRepository
	testutils.EmptyTestRepository
}

func setup(t *testing.T, ctrl *gomock.Controller) (*mock_handler.MockContext, *mock_channel.MockManager, *Repo) {
	handlerCtx := mock_handler.NewMockContext(ctrl)
	cm := mock_channel.NewMockManager(ctrl)

	repo := &Repo{
		MockTagRepository:  mock_repository.NewMockTagRepository(ctrl),
		MockUserRepository: mock_repository.NewMockUserRepository(ctrl),
		MockBotRepository:  mock_repository.NewMockBotRepository(ctrl),
	}

	handlerCtx.EXPECT().
		CM().
		Return(cm).
		AnyTimes()
	handlerCtx.EXPECT().
		L().
		Return(zap.NewNop()).
		AnyTimes()
	handlerCtx.EXPECT().
		R().
		Return(repo).
		AnyTimes()
	return handlerCtx, cm, repo
}

func registerBot(t *testing.T, handlerCtx *mock_handler.MockContext, b *model.Bot) {
	handlerCtx.EXPECT().
		GetBot(b.ID).
		Return(b, nil).
		AnyTimes()
	handlerCtx.EXPECT().
		GetBotByBotUserID(b.BotUserID).
		Return(b, nil).
		AnyTimes()
	for event := range b.SubscribeEvents {
		handlerCtx.EXPECT().
			GetBots(event).
			Return([]*model.Bot{b}, nil).
			AnyTimes()
	}
}

func registerUser(repo *Repo, u *model.User) {
	repo.MockUserRepository.EXPECT().
		GetUser(u.ID, gomock.Any()).
		Return(u, nil).
		AnyTimes()
}

func registerChannel(cm *mock_channel.MockManager, ch *model.Channel) {
	cm.EXPECT().
		GetChannel(ch.ID).
		Return(ch, nil).
		AnyTimes()
}

func registerTag(repo *Repo, t *model.Tag) {
	repo.MockTagRepository.EXPECT().
		GetTagByID(t.ID).
		Return(t, nil).
		AnyTimes()
}

func expectMulticast(handlerCtx *mock_handler.MockContext, ev model.BotEventType, payload interface{}, targets []*model.Bot) {
	handlerCtx.EXPECT().
		Multicast(ev, payload, targets).
		Times(1)
}

func expectUnicast(handlerCtx *mock_handler.MockContext, ev model.BotEventType, payload interface{}, target *model.Bot) {
	handlerCtx.EXPECT().
		Unicast(ev, payload, target).
		Times(1)
}
