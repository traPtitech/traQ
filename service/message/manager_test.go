package message

import (
	"github.com/golang/mock/gomock"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/testutils"
)

type Repo struct {
	*mock_repository.MockChannelRepository
	*mock_repository.MockMessageRepository
	*mock_repository.MockPinRepository
	testutils.EmptyTestRepository
}

func NewMockRepo(ctrl *gomock.Controller) *Repo {
	return &Repo{
		MockChannelRepository: mock_repository.NewMockChannelRepository(ctrl),
		MockMessageRepository: mock_repository.NewMockMessageRepository(ctrl),
		MockPinRepository:     mock_repository.NewMockPinRepository(ctrl),
	}
}
