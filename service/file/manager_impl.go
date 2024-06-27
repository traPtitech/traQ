package file

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/storage"
)

type managerImpl struct {
	repo repository.FileRepository
	fs   storage.FileStorage
	ip   imaging.Processor
	l    *zap.Logger
}

func makeSureSeekable(r io.Reader) (io.ReadSeeker, error) {
	src, ok := r.(io.ReadSeeker)
	if ok {
		return src, nil
	}
	// Seek出来ないと困るので全読み込み
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read whole src stream: %w", err)
	}
	return bytes.NewReader(b), nil
}

func InitFileManager(repo repository.FileRepository, fs storage.FileStorage, ip imaging.Processor, l *zap.Logger) (Manager, error) {
	return &managerImpl{
		repo: repo,
		fs:   fs,
		ip:   ip,
		l:    l.Named("file_manager"),
	}, nil
}

func (m *managerImpl) canGenerateThumbnail(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func (m *managerImpl) canGenerateWaveform(mimeType string) bool {
	switch mimeType {
	case "audio/mpeg", "audio/mp3", "audio/wav", "audio/x-wav":
		return true
	default:
		return false
	}
}

func (m *managerImpl) Save(args SaveArgs) (model.File, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	f := &model.FileMeta{
		ID:              uuid.Must(uuid.NewV7()),
		Name:            args.FileName,
		Mime:            args.MimeType,
		Size:            args.FileSize,
		CreatorID:       args.CreatorID,
		Type:            args.FileType,
		ChannelID:       args.ChannelID,
		IsAnimatedImage: false,
	}

	// アニメーション画像判定
	switch args.MimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		src, err := makeSureSeekable(args.Src)
		if err != nil {
			return nil, err
		}
		args.Src = src

		if isAnimated, err := isAnimatedImage(src); isAnimated && err == nil {
			f.IsAnimatedImage = true
		}

		// ストリームを先頭に戻す
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek src stream: %w", err)
		}
	}

	// サムネイル画像生成
	if args.Thumbnail == nil && m.canGenerateThumbnail(args.MimeType) {
		src, err := makeSureSeekable(args.Src)
		if err != nil {
			return nil, err
		}
		args.Src = src

		thumb, err := m.ip.Thumbnail(src)
		if err != nil {
			m.l.Warn("failed to generate thumbnail", zap.Error(err), zap.Stringer("fid", f.ID))
		} else {
			args.Thumbnail = thumb
		}

		// ストリームを先頭に戻す
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek src stream: %w", err)
		}
	}

	// 波形画像生成
	if m.canGenerateWaveform(args.MimeType) {
		src, err := makeSureSeekable(args.Src)
		if err != nil {
			return nil, err
		}
		args.Src = src

		const (
			waveformWidth  = 1280
			waveformHeight = 540
		)

		var r io.Reader
		switch args.MimeType {
		case "audio/mpeg", "audio/mp3":
			r, err = m.ip.WaveformMp3(src, waveformWidth, waveformHeight)
			if err != nil {
				m.l.Warn("failed to generate thumbnail", zap.Error(err), zap.Stringer("fid", f.ID))
			}
		case "audio/wav", "audio/x-wav":
			r, err = m.ip.WaveformWav(src, waveformWidth, waveformHeight)
			if err != nil {
				m.l.Warn("failed to generate thumbnail", zap.Error(err), zap.Stringer("fid", f.ID))
			}
		}

		if r != nil {
			thumbnail := model.FileThumbnail{
				Type:   model.ThumbnailTypeWaveform,
				Mime:   "image/svg+xml",
				Width:  waveformWidth,
				Height: waveformHeight,
			}
			f.Thumbnails = append(f.Thumbnails, thumbnail)

			key := f.ID.String() + "-" + model.ThumbnailTypeWaveform.Suffix()
			if err := m.fs.SaveByKey(r, key, key+".svg", "image/svg+xml", model.FileTypeThumbnail); err != nil {
				return nil, fmt.Errorf("failed to save thumbnail to storage: %w", err)
			}
		}

		// ストリームを先頭に戻す
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek src stream: %w", err)
		}
	}

	if args.Thumbnail != nil {
		thumbnail := model.FileThumbnail{
			Type:   model.ThumbnailTypeImage,
			Mime:   "image/png",
			Width:  args.Thumbnail.Bounds().Size().X,
			Height: args.Thumbnail.Bounds().Size().Y,
		}
		f.Thumbnails = append(f.Thumbnails, thumbnail)

		r, w := io.Pipe()
		go func() {
			defer w.Close()
			_ = png.Encode(w, args.Thumbnail)
		}()

		key := f.ID.String() + "-" + model.ThumbnailTypeImage.Suffix()
		if err := m.fs.SaveByKey(r, key, key+".png", "image/png", model.FileTypeThumbnail); err != nil {
			return nil, fmt.Errorf("failed to save thumbnail to storage: %w", err)
		}
	}

	hash := md5.New()
	if err := m.fs.SaveByKey(io.TeeReader(args.Src, hash), f.ID.String(), f.Name, f.Mime, f.Type); err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}
	f.Hash = hex.EncodeToString(hash.Sum(nil))

	var acl []*model.FileACLEntry
	for uid, allow := range args.ACL {
		acl = append(acl, &model.FileACLEntry{
			UserID: uid,
			Allow:  allow,
		})
	}

	err := m.repo.SaveFileMeta(f, acl)
	if err != nil {
		if err := m.fs.DeleteByKey(f.ID.String(), f.Type); err != nil {
			m.l.Warn("failed to delete file from storage during rollback", zap.Error(err), zap.Stringer("fid", f.ID))
		}
		for _, t := range f.Thumbnails {
			if err := m.fs.DeleteByKey(f.ID.String()+"-"+t.Type.Suffix(), model.FileTypeThumbnail); err != nil {
				m.l.Warn("failed to delete thumbnail from storage during rollback", zap.Error(err), zap.Stringer("fid", f.ID))
			}
		}
		return nil, fmt.Errorf("failed to SaveFileMeta: %w", err)
	}
	return m.makeFileMeta(f), nil
}

func (m *managerImpl) Get(id uuid.UUID) (model.File, error) {
	meta, err := m.repo.GetFileMeta(id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to GetFileMeta: %w", err)
	}
	return m.makeFileMeta(meta), nil
}

func (m *managerImpl) List(q repository.FilesQuery) ([]model.File, bool, error) {
	r, more, err := m.repo.GetFileMetas(q)
	if err != nil {
		return nil, false, fmt.Errorf("failed to GetFileMetas: %w", err)
	}
	return m.makeFileMetas(r), more, nil
}

func (m *managerImpl) Delete(id uuid.UUID) error {
	meta, err := m.repo.GetFileMeta(id)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to GetFileMeta: %w", err)
	}

	if err := m.repo.DeleteFileMeta(id); err != nil {
		return fmt.Errorf("failed to DeleteFileMeta: %w", err)
	}
	if err := m.fs.DeleteByKey(meta.ID.String(), meta.Type); err != nil {
		m.l.Warn("failed to delete file from storage", zap.Error(err), zap.Stringer("fid", meta.ID))
	}
	for _, t := range meta.Thumbnails {
		if err := m.fs.DeleteByKey(meta.ID.String()+"-"+t.Type.Suffix(), model.FileTypeThumbnail); err != nil {
			m.l.Warn("failed to delete thumbnail from storage", zap.Error(err), zap.Stringer("fid", meta.ID))
		}
	}
	return nil
}

func (m *managerImpl) Accessible(fileID, userID uuid.UUID) (bool, error) {
	ok, err := m.repo.IsFileAccessible(fileID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to IsFileAccessible: %w", err)
	}
	return ok, nil
}

func (m *managerImpl) makeFileMeta(f *model.FileMeta) model.File {
	return &fileMetaImpl{meta: f, fs: m.fs}
}

func (m *managerImpl) makeFileMetas(fs []*model.FileMeta) []model.File {
	result := make([]model.File, len(fs))
	for i, f := range fs {
		result[i] = m.makeFileMeta(f)
	}
	return result
}
