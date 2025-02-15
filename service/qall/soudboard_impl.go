package qall

import (
	"io"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
)

type soundboardManagerImpl struct {
	repo repository.SoundboardRepository
	fs   storage.FileStorage
	l    *zap.Logger
}

func NewSoundboardManager(repo repository.SoundboardRepository, fs storage.FileStorage, logger *zap.Logger) (soundboardManagerImpl, error) {
	return soundboardManagerImpl{
		repo: repo,
		fs:   fs,
		l:    logger.Named("soundboard_manager"),
	}, nil
}

func (m *soundboardManagerImpl) CreateSoundBoardItem(soundID uuid.UUID, contentType string, fileType model.FileType, src io.Reader) error {
	err := m.fs.SaveByKey(src, soundID.String(), "soundboardItem", contentType, fileType)
	if err != nil {
		m.l.Error("failed to save soundboard item", zap.Error(err))
		return err
	}

	return m.repo.CreateSoundBoardItem(soundID, "soundboardItem", soundID, soundID)
}

func (m *soundboardManagerImpl) GetURL(soundID uuid.UUID) (string, error) {
	return m.fs.GenerateAccessURL(soundID.String(), model.FileTypeSoundboardItem)
}

func (m *soundboardManagerImpl) DeleteSoundBoardItem(soundID uuid.UUID) error {
	err := m.fs.DeleteByKey(soundID.String(), model.FileTypeSoundboardItem)
	if err != nil {
		m.l.Error("failed to delete soundboard item", zap.Error(err))
		return err
	}

	return m.repo.DeleteSoundboardItem(soundID)
}
