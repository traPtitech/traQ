package file

import (
	"bytes"
	"errors"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/imaging/mock_imaging"
	imaging2 "github.com/traPtitech/traQ/utils/imaging"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/storage"
	"github.com/traPtitech/traQ/utils/storage/mock_storage"
)

var errMock = errors.New("mock error")

func initFM(_ *testing.T, repo repository.FileRepository, fs storage.FileStorage, ip imaging.Processor) *managerImpl {
	return &managerImpl{
		repo: repo,
		fs:   fs,
		ip:   ip,
		l:    zap.NewNop(),
	}
}

func TestManagerImpl_Save(t *testing.T) {
	t.Parallel()

	t.Run("text file", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		fm := initFM(t, repo, fs, nil)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		args := SaveArgs{
			FileName:  "test.txt",
			FileSize:  int64(len(data)),
			MimeType:  "text/plain",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewReader(data),
		}

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, _ = io.Copy(io.Discard, src)
				return nil
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			DoAndReturn(func(meta *model.FileMeta, _ []*model.FileACLEntry) error {
				meta.CreatedAt = time.Now()
				return nil
			}).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			assert.EqualValues(t, 0, len(result.GetThumbnails()))
		}
	})

	t.Run("file with thumbnail", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		fm := initFM(t, repo, fs, nil)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		thumb := imaging2.GenerateIcon("test")
		args := SaveArgs{
			FileName:  "dummy.png",
			FileSize:  int64(len(data)),
			MimeType:  "image/png",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewReader(data),
			Thumbnail: thumb,
		}

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, _ = io.Copy(io.Discard, src)
				return nil
			}).
			Times(1)
		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), gomock.Any(), "image/png", model.FileTypeThumbnail).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, err := png.Decode(src)
				return err
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			DoAndReturn(func(meta *model.FileMeta, _ []*model.FileACLEntry) error {
				meta.CreatedAt = time.Now()
				return nil
			}).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			thumbs := result.GetThumbnails()
			assert.EqualValues(t, 1, len(thumbs))
			assert.EqualValues(t, model.ThumbnailTypeImage, thumbs[0].Type)
			assert.EqualValues(t, "image/png", thumbs[0].Mime)
			assert.EqualValues(t, thumb.Bounds().Size().X, thumbs[0].Width)
			assert.EqualValues(t, thumb.Bounds().Size().Y, thumbs[0].Height)
		}
	})

	t.Run("image with generating thumbnail (io.ReadSeeker)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		ip := mock_imaging.NewMockProcessor(ctrl)
		fm := initFM(t, repo, fs, ip)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		thumb := imaging2.GenerateIcon("test")
		args := SaveArgs{
			FileName:  "dummy.png",
			FileSize:  int64(len(data)),
			MimeType:  "image/png",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewReader(data),
		}

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			Do(func(src io.Reader, _, _, _ string, _ model.FileType) {
				_, _ = io.Copy(io.Discard, src)
			}).
			Return(nil).
			Times(1)
		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), gomock.Any(), "image/png", model.FileTypeThumbnail).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, err := png.Decode(src)
				return err
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			Do(func(meta *model.FileMeta, _ []*model.FileACLEntry) { meta.CreatedAt = time.Now() }).
			Return(nil).
			Times(1)
		ip.EXPECT().
			Thumbnail(gomock.Any()).
			Do(func(src io.ReadSeeker) { _, _ = io.Copy(io.Discard, src) }).
			Return(thumb, nil).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			thumbs := result.GetThumbnails()
			assert.EqualValues(t, 1, len(thumbs))
			assert.EqualValues(t, model.ThumbnailTypeImage, thumbs[0].Type)
			assert.EqualValues(t, "image/png", thumbs[0].Mime)
			assert.EqualValues(t, thumb.Bounds().Size().X, thumbs[0].Width)
			assert.EqualValues(t, thumb.Bounds().Size().Y, thumbs[0].Height)
		}
	})

	t.Run("image with generating thumbnail (io.Reader)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		ip := mock_imaging.NewMockProcessor(ctrl)
		fm := initFM(t, repo, fs, ip)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		thumb := imaging2.GenerateIcon("test")
		args := SaveArgs{
			FileName:  "dummy.png",
			FileSize:  int64(len(data)),
			MimeType:  "image/png",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewBuffer(data),
		}

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			Do(func(src io.Reader, _, _, _ string, _ model.FileType) {
				_, _ = io.Copy(io.Discard, src)
			}).
			Return(nil).
			Times(1)
		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), gomock.Any(), "image/png", model.FileTypeThumbnail).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, err := png.Decode(src)
				return err
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			Do(func(meta *model.FileMeta, _ []*model.FileACLEntry) { meta.CreatedAt = time.Now() }).
			Return(nil).
			Times(1)
		ip.EXPECT().
			Thumbnail(gomock.Any()).
			Do(func(src io.ReadSeeker) { _, _ = io.Copy(io.Discard, src) }).
			Return(thumb, nil).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			thumbs := result.GetThumbnails()
			assert.EqualValues(t, 1, len(thumbs))
			assert.EqualValues(t, model.ThumbnailTypeImage, thumbs[0].Type)
			assert.EqualValues(t, "image/png", thumbs[0].Mime)
			assert.EqualValues(t, thumb.Bounds().Size().X, thumbs[0].Width)
			assert.EqualValues(t, thumb.Bounds().Size().Y, thumbs[0].Height)
		}
	})

	t.Run("audio with generating waveform (mp3, io.ReadSeeker)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		ip := mock_imaging.NewMockProcessor(ctrl)
		fm := initFM(t, repo, fs, ip)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		args := SaveArgs{
			FileName:  "dummy.mp3",
			FileSize:  int64(len(data)),
			MimeType:  "audio/mp3",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewReader(data),
		}
		waveform := bytes.NewBufferString("dummy svg file")

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			Do(func(src io.Reader, _, _, _ string, _ model.FileType) {
				_, _ = io.Copy(io.Discard, src)
			}).
			Return(nil).
			Times(1)
		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), gomock.Any(), "image/svg+xml", model.FileTypeThumbnail).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, _ = io.Copy(io.Discard, src)
				return nil
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			Do(func(meta *model.FileMeta, _ []*model.FileACLEntry) { meta.CreatedAt = time.Now() }).
			Return(nil).
			Times(1)
		ip.EXPECT().
			WaveformMp3(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(src io.ReadSeeker, _, _ int) { _, _ = io.Copy(io.Discard, src) }).
			Return(waveform, nil).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			thumbs := result.GetThumbnails()
			assert.EqualValues(t, 1, len(thumbs))
			assert.EqualValues(t, model.ThumbnailTypeWaveform, thumbs[0].Type)
			assert.EqualValues(t, "image/svg+xml", thumbs[0].Mime)
		}
	})

	t.Run("audio with generating waveform (wav, io.ReadSeeker)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fs := mock_storage.NewMockFileStorage(ctrl)
		ip := mock_imaging.NewMockProcessor(ctrl)
		fm := initFM(t, repo, fs, ip)

		data := []byte("test text file")
		hash := "7e6d5d7ae4965bfecc6d818f76eb832b"
		args := SaveArgs{
			FileName:  "dummy.wav",
			FileSize:  int64(len(data)),
			MimeType:  "audio/wav",
			FileType:  model.FileTypeUserFile,
			ChannelID: optional.From(uuid.NewV3(uuid.Nil, "c")),
			Src:       bytes.NewReader(data),
		}
		waveform := bytes.NewBufferString("dummy svg file")

		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), args.FileName, args.MimeType, args.FileType).
			Do(func(src io.Reader, _, _, _ string, _ model.FileType) {
				_, _ = io.Copy(io.Discard, src)
			}).
			Return(nil).
			Times(1)
		fs.EXPECT().
			SaveByKey(gomock.Any(), gomock.Any(), gomock.Any(), "image/svg+xml", model.FileTypeThumbnail).
			DoAndReturn(func(src io.Reader, _, _, _ string, _ model.FileType) error {
				_, _ = io.Copy(io.Discard, src)
				return nil
			}).
			Times(1)
		repo.EXPECT().
			SaveFileMeta(gomock.Any(), []*model.FileACLEntry{{UserID: uuid.Nil, Allow: true}}).
			Do(func(meta *model.FileMeta, _ []*model.FileACLEntry) { meta.CreatedAt = time.Now() }).
			Return(nil).
			Times(1)
		ip.EXPECT().
			WaveformWav(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(src io.ReadSeeker, _, _ int) { _, _ = io.Copy(io.Discard, src) }).
			Return(waveform, nil).
			Times(1)

		result, err := fm.Save(args)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, result.GetID())
			assert.EqualValues(t, args.FileName, result.GetFileName())
			assert.EqualValues(t, args.FileSize, result.GetFileSize())
			assert.EqualValues(t, args.MimeType, result.GetMIMEType())
			assert.EqualValues(t, args.FileType, result.GetFileType())
			assert.EqualValues(t, args.ChannelID, result.GetUploadChannelID())
			assert.EqualValues(t, args.CreatorID, result.GetCreatorID())
			assert.EqualValues(t, hash, result.GetMD5Hash())
			assert.EqualValues(t, false, result.IsAnimatedImage())
			assert.NotEmpty(t, result.GetCreatedAt())
			thumbs := result.GetThumbnails()
			assert.EqualValues(t, 1, len(thumbs))
			assert.EqualValues(t, model.ThumbnailTypeWaveform, thumbs[0].Type)
			assert.EqualValues(t, "image/svg+xml", thumbs[0].Mime)
		}
	})
}

func TestManagerImpl_Get(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		meta := &model.FileMeta{
			ID:        uuid.NewV3(uuid.Nil, "f1"),
			Name:      "file",
			Mime:      "text/plain",
			Size:      10,
			Hash:      "d41d8cd98f00b204e9800998ecf8427e",
			Type:      model.FileTypeUserFile,
			CreatedAt: time.Now(),
		}
		meta.Thumbnails = []model.FileThumbnail{{
			FileID: meta.ID,
			Type:   model.ThumbnailTypeImage,
		}}

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
			Return(nil, errMock).
			Times(1)

		_, err := fm.Get(uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, errMock, errors.Unwrap(err))
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
			ID:        uuid.NewV3(uuid.Nil, "f1"),
			Name:      "file",
			Mime:      "text/plain",
			Size:      10,
			Hash:      "d41d8cd98f00b204e9800998ecf8427e",
			Type:      model.FileTypeUserFile,
			CreatedAt: time.Now(),
		}
		meta.Thumbnails = []model.FileThumbnail{{
			FileID: meta.ID,
			Type:   model.ThumbnailTypeImage,
		}}

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
			Return(nil, false, errMock).
			Times(1)

		arr, more, err := fm.List(repository.FilesQuery{})
		if assert.Error(t, err) {
			assert.Nil(t, arr)
			assert.False(t, more)
			assert.Equal(t, errMock, errors.Unwrap(err))
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
			ID:        uuid.NewV3(uuid.Nil, "f1"),
			Name:      "file",
			Mime:      "text/plain",
			Size:      10,
			Hash:      "d41d8cd98f00b204e9800998ecf8427e",
			Type:      model.FileTypeUserFile,
			CreatedAt: time.Now(),
		}
		meta.Thumbnails = []model.FileThumbnail{
			{
				FileID: meta.ID,
				Type:   model.ThumbnailTypeImage,
			},
			{
				FileID: meta.ID,
				Type:   model.ThumbnailTypeWaveform,
			},
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
			DeleteByKey(meta.ID.String()+"-"+model.ThumbnailTypeImage.Suffix(), model.FileTypeThumbnail).
			Return(nil).
			Times(1)
		fs.EXPECT().
			DeleteByKey(meta.ID.String()+"-"+model.ThumbnailTypeWaveform.Suffix(), model.FileTypeThumbnail).
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
			ID:        uuid.NewV3(uuid.Nil, "f1"),
			Name:      "file",
			Mime:      "text/plain",
			Size:      10,
			Hash:      "d41d8cd98f00b204e9800998ecf8427e",
			Type:      model.FileTypeUserFile,
			CreatedAt: time.Now(),
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
			Return(nil, errMock).
			Times(1)

		err := fm.Delete(uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, errMock, errors.Unwrap(err))
		}
	})

	t.Run("repo error 2", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockFileRepository(ctrl)
		fm := initFM(t, repo, nil, nil)

		meta := &model.FileMeta{
			ID:        uuid.NewV3(uuid.Nil, "f1"),
			Name:      "file",
			Mime:      "text/plain",
			Size:      10,
			Hash:      "d41d8cd98f00b204e9800998ecf8427e",
			Type:      model.FileTypeUserFile,
			CreatedAt: time.Now(),
		}

		repo.EXPECT().
			GetFileMeta(meta.ID).
			Return(meta, nil).
			Times(1)
		repo.EXPECT().
			DeleteFileMeta(meta.ID).
			Return(errMock).
			Times(1)

		err := fm.Delete(meta.ID)
		if assert.Error(t, err) {
			assert.Equal(t, errMock, errors.Unwrap(err))
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
		repo.EXPECT().IsFileAccessible(fid, uid).Return(false, errMock).Times(1)

		ok, err := fm.Accessible(fid, uid)
		if assert.Error(t, err) {
			assert.False(t, ok)
			assert.Equal(t, errMock, errors.Unwrap(err))
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
