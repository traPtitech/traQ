// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
package utils

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/utils/message"
)

type replaceMapperImpl struct {
	repo repository.Repository
	cm   channel.Manager
}

func (m *replaceMapperImpl) Channel(path string) (uuid.UUID, bool) {
	id := m.cm.PublicChannelTree().GetChannelIDFromPath(path)
	return id, id != uuid.Nil
}

func (m *replaceMapperImpl) Group(name string) (uuid.UUID, bool) {
	g, err := m.repo.GetUserGroupByName(name)
	if err != nil {
		return uuid.Nil, false
	}
	return g.ID, true
}

func (m *replaceMapperImpl) User(name string) (uuid.UUID, bool) {
	u, err := m.repo.GetUserByName(name, false)
	if err != nil {
		return uuid.Nil, false
	}
	return u.GetID(), true
}

func NewReplaceMapper(repo repository.Repository, cm channel.Manager) message.ReplaceMapper {
	return &replaceMapperImpl{
		repo: repo,
		cm:   cm,
	}
}
