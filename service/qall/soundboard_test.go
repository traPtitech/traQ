package qall

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/utils/storage/mock_storage"
	"go.uber.org/zap"
)

func setupSoundboardTest(t *testing.T) (*gomock.Controller, *mock_repository.MockSoundboardRepository, *mock_storage.MockFileStorage, *hub.Hub, *soundboardManager) {
	ctrl := gomock.NewController(t)
	mockRepo := mock_repository.NewMockSoundboardRepository(ctrl)
	mockStorage := mock_storage.NewMockFileStorage(ctrl)
	h := hub.New()
	logger := zap.NewNop()

	manager, _ := NewSoundboardManager(mockRepo, mockStorage, logger, h)
	impl, ok := manager.(*soundboardManager)
	assert.True(t, ok)

	return ctrl, mockRepo, mockStorage, h, impl
}

func TestSoundboardManager_GetURL(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl, _, mockStorage, _, manager := setupSoundboardTest(t)
		defer ctrl.Finish()

		soundID := uuid.Must(uuid.NewV7())
		expectedURL := "https://example.com/soundboard/123.mp3"

		// GenerateAccessURLが呼ばれることを期待
		mockStorage.EXPECT().
			GenerateAccessURL(soundID.String(), model.FileTypeSoundboardItem).
			Return(expectedURL, nil).
			Times(1)

		// テスト実行
		url, err := manager.GetURL(soundID)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, url)
	})

	t.Run("storage_error", func(t *testing.T) {
		t.Parallel()
		ctrl, _, mockStorage, _, manager := setupSoundboardTest(t)
		defer ctrl.Finish()

		soundID := uuid.Must(uuid.NewV7())
		mockErr := errors.New("storage error")

		mockStorage.EXPECT().
			GenerateAccessURL(soundID.String(), model.FileTypeSoundboardItem).
			Return("", mockErr).
			Times(1)

		// テスト実行
		_, err := manager.GetURL(soundID)
		assert.Error(t, err)
		assert.Equal(t, mockErr, err)
	})
}

func TestSoundboardManager_DeleteSoundboardItem(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl, mockRepo, mockStorage, _, manager := setupSoundboardTest(t)
		defer ctrl.Finish()

		soundID := uuid.Must(uuid.NewV7())

		// DeleteByKeyが呼ばれることを期待
		mockStorage.EXPECT().
			DeleteByKey(soundID.String(), model.FileTypeSoundboardItem).
			Return(nil).
			Times(1)

		// DeleteSoundboardItemが呼ばれることを期待
		mockRepo.EXPECT().
			DeleteSoundboardItem(soundID).
			Return(nil).
			Times(1)

		// テスト実行
		err := manager.DeleteSoundboardItem(soundID)
		assert.NoError(t, err)
	})

	t.Run("repository_error", func(t *testing.T) {
		t.Parallel()
		ctrl, mockRepo, mockStorage, _, manager := setupSoundboardTest(t)
		defer ctrl.Finish()

		soundID := uuid.Must(uuid.NewV7())
		mockErr := errors.New("repository error")

		mockStorage.EXPECT().
			DeleteByKey(soundID.String(), model.FileTypeSoundboardItem).
			Return(nil).
			Times(1)

		mockRepo.EXPECT().
			DeleteSoundboardItem(soundID).
			Return(mockErr).
			Times(1)

		// テスト実行
		err := manager.DeleteSoundboardItem(soundID)
		assert.Error(t, err)
		assert.Equal(t, mockErr, err)
	})

	t.Run("storage_error", func(t *testing.T) {
		t.Parallel()
		ctrl, _, mockStorage, _, manager := setupSoundboardTest(t)
		defer ctrl.Finish()

		soundID := uuid.Must(uuid.NewV7())
		mockErr := errors.New("storage error")

		mockStorage.EXPECT().
			DeleteByKey(soundID.String(), model.FileTypeSoundboardItem).
			Return(mockErr).
			Times(1)

		// テスト実行
		err := manager.DeleteSoundboardItem(soundID)
		assert.Error(t, err)
		assert.Equal(t, mockErr, err)
	})
}

func TestNewSoundboardManager(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockRepo := mock_repository.NewMockSoundboardRepository(ctrl)
		mockStorage := mock_storage.NewMockFileStorage(ctrl)
		h := hub.New()
		logger := zap.NewNop()

		// テスト実行
		manager, err := NewSoundboardManager(mockRepo, mockStorage, logger, h)

		// 検証
		assert.NoError(t, err)
		assert.NotNil(t, manager)

		// 型アサーションを行い、適切な型が返されていることを確認
		_, ok := manager.(*soundboardManager)
		assert.True(t, ok)
	})
}
