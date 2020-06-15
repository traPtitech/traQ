package file

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/storage"
	"github.com/traPtitech/traQ/utils/storage/mock_storage"
	"go.uber.org/zap"
	"testing"
	"time"
)

var mockErr = errors.New("mock error")

func initFM(t *testing.T, repo repository.FileRepository, fs storage.FileStorage, ip imaging.Processor) *managerImpl {
	return &managerImpl{
		repo: repo,
		fs:   fs,
		ip:   ip,
		l:    zap.NewNop(),
	}
}

func TestManagerImpl_Get(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		meta := &model.FileMeta{
			ID:           uuid.NewV3(uuid.Nil, "f1"),
			Name:         "file",
			Mime:         "text/plain",
			Size:         10,
			Hash:         "d41d8cd98f00b204e9800998ecf8427e",
			Type:         model.FileTypeUserFile,
			HasThumbnail: true,
			CreatedAt:    time.Now(),
		}

		repo.EXPECT().
			GetFileMeta(meta.ID).
			Return(meta, nil).
			Times(1)

		res, err := fm.Get(meta.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, meta.ID, res.GetID())
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		repo.EXPECT().
			GetFileMeta(uuid.Nil).
			Return(nil, repository.ErrNotFound).
			Times(1)

		_, err := fm.Get(uuid.Nil)
		if assert.Error(t, err) {
			assert.EqualError(t, ErrNotFound, err.Error())
		}
	})

	t.Run("repo error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		repo.EXPECT().
			GetFileMeta(uuid.Nil).
			Return(nil, mockErr).
			Times(1)

		_, err := fm.Get(uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_List(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		meta := &model.FileMeta{
			ID:           uuid.NewV3(uuid.Nil, "f1"),
			Name:         "file",
			Mime:         "text/plain",
			Size:         10,
			Hash:         "d41d8cd98f00b204e9800998ecf8427e",
			Type:         model.FileTypeUserFile,
			HasThumbnail: true,
			CreatedAt:    time.Now(),
		}

		repo.EXPECT().
			GetFileMetas(gomock.Any()).
			Return([]*model.FileMeta{meta, meta, meta}, true, nil).
			Times(1)

		res, more, err := fm.List(repository.FilesQuery{})
		if assert.NoError(t, err) {
			assert.True(t, more)
			assert.Len(t, res, 3)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		repo.EXPECT().
			GetFileMetas(gomock.Any()).
			Return(nil, false, mockErr).
			Times(1)

		arr, more, err := fm.List(repository.FilesQuery{})
		if assert.Error(t, err) {
			assert.Nil(t, arr)
			assert.False(t, more)
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_Delete(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		fm := initFM(t, repo, fs, nil)

		meta := &model.FileMeta{
			ID:           uuid.NewV3(uuid.Nil, "f1"),
			Name:         "file",
			Mime:         "text/plain",
			Size:         10,
			Hash:         "d41d8cd98f00b204e9800998ecf8427e",
			Type:         model.FileTypeUserFile,
			HasThumbnail: true,
			CreatedAt:    time.Now(),
		}

		repo.EXPECT().
			GetFileMeta(meta.ID).
			Return(meta, nil).
			Times(1)
		repo.EXPECT().
			DeleteFileMeta(meta.ID).
			Return(nil).
			Times(1)
		fs.EXPECT().
			DeleteByKey(meta.ID.String(), meta.Type).
			Return(nil).
			Times(1)
		fs.EXPECT().
			DeleteByKey(meta.ID.String()+"-thumb", model.FileTypeThumbnail).
			Return(nil).
			Times(1)

		assert.NoError(t, fm.Delete(meta.ID))
	})

	t.Run("success (no thumbnail)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		fm := initFM(t, repo, fs, nil)

		meta := &model.FileMeta{
			ID:           uuid.NewV3(uuid.Nil, "f1"),
			Name:         "file",
			Mime:         "text/plain",
			Size:         10,
			Hash:         "d41d8cd98f00b204e9800998ecf8427e",
			Type:         model.FileTypeUserFile,
			HasThumbnail: false,
			CreatedAt:    time.Now(),
		}

		repo.EXPECT().
			GetFileMeta(meta.ID).
			Return(meta, nil).
			Times(1)
		repo.EXPECT().
			DeleteFileMeta(meta.ID).
			Return(nil).
			Times(1)
		fs.EXPECT().
			DeleteByKey(meta.ID.String(), meta.Type).
			Return(nil).
			Times(1)

		assert.NoError(t, fm.Delete(meta.ID))
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		repo.EXPECT().
			GetFileMeta(uuid.Nil).
			Return(nil, repository.ErrNotFound).
			Times(1)

		assert.EqualError(t, ErrNotFound, fm.Delete(uuid.Nil).Error())
	})

	t.Run("repo error 1", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		repo.EXPECT().
			GetFileMeta(uuid.Nil).
			Return(nil, mockErr).
			Times(1)

		err := fm.Delete(uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("repo error 2", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		meta := &model.FileMeta{
			ID:           uuid.NewV3(uuid.Nil, "f1"),
			Name:         "file",
			Mime:         "text/plain",
			Size:         10,
			Hash:         "d41d8cd98f00b204e9800998ecf8427e",
			Type:         model.FileTypeUserFile,
			HasThumbnail: false,
			CreatedAt:    time.Now(),
		}

		repo.EXPECT().
			GetFileMeta(meta.ID).
			Return(meta, nil).
			Times(1)
		repo.EXPECT().
			DeleteFileMeta(meta.ID).
			Return(mockErr).
			Times(1)

		err := fm.Delete(meta.ID)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_Accessible(t *testing.T) {
	t.Parallel()

	t.Run("repo error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		fid := uuid.NewV3(uuid.Nil, "f1")
		uid := uuid.NewV3(uuid.Nil, "u1")
		repo.EXPECT().IsFileAccessible(fid, uid).Return(false, mockErr).Times(1)

		ok, err := fm.Accessible(fid, uid)
		if assert.Error(t, err) {
			assert.False(t, ok)
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("success (true)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		fid := uuid.NewV3(uuid.Nil, "f1")
		uid := uuid.NewV3(uuid.Nil, "u1")
		repo.EXPECT().IsFileAccessible(fid, uid).Return(true, nil).Times(1)

		ok, err := fm.Accessible(fid, uid)
		if assert.NoError(t, err) {
			assert.True(t, ok)
		}
	})

	t.Run("success (false)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		fid := uuid.NewV3(uuid.Nil, "f1")
		uid := uuid.NewV3(uuid.Nil, "u1")
		repo.EXPECT().IsFileAccessible(fid, uid).Return(false, nil).Times(1)

		ok, err := fm.Accessible(fid, uid)
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
}
