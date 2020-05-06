package repository

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/ioext"
	"github.com/traPtitech/traQ/utils/storage"
	"gopkg.in/guregu/null.v3"
	"image/png"
	"io"
	"time"
)

type fileImpl struct {
	FS storage.FileStorage
}

type fileMetaImpl struct {
	meta *model.File
	fs   storage.FileStorage
}

func (f *fileMetaImpl) GetID() uuid.UUID {
	return f.meta.ID
}

func (f *fileMetaImpl) GetFileName() string {
	return f.meta.Name
}

func (f *fileMetaImpl) GetMIMEType() string {
	return f.meta.Mime
}

func (f *fileMetaImpl) GetFileSize() int64 {
	return f.meta.Size
}

func (f *fileMetaImpl) GetFileType() model.FileType {
	return f.meta.Type
}

func (f *fileMetaImpl) GetCreatorID() uuid.NullUUID {
	return f.meta.CreatorID
}

func (f *fileMetaImpl) GetMD5Hash() string {
	return f.meta.Hash
}

func (f *fileMetaImpl) HasThumbnail() bool {
	return f.meta.HasThumbnail
}

func (f *fileMetaImpl) GetThumbnailMIMEType() string {
	return f.meta.ThumbnailMime.String
}

func (f *fileMetaImpl) GetThumbnailWidth() int {
	return f.meta.ThumbnailWidth
}

func (f *fileMetaImpl) GetThumbnailHeight() int {
	return f.meta.ThumbnailHeight
}

func (f *fileMetaImpl) GetUploadChannelID() uuid.NullUUID {
	return f.meta.ChannelID
}

func (f *fileMetaImpl) GetCreatedAt() time.Time {
	return f.meta.CreatedAt
}

func (f *fileMetaImpl) Open() (ioext.ReadSeekCloser, error) {
	return f.fs.OpenFileByKey(f.GetID().String(), f.GetFileType())
}

func (f *fileMetaImpl) OpenThumbnail() (ioext.ReadSeekCloser, error) {
	if !f.HasThumbnail() {
		return nil, ErrNotFound
	}
	return f.fs.OpenFileByKey(f.GetID().String()+"-thumb", model.FileTypeThumbnail)
}

func (f *fileMetaImpl) GetAlternativeURL() string {
	url, _ := f.fs.GenerateAccessURL(f.GetID().String(), f.GetFileType())
	return url
}

// GetFiles implements FileRepository interface.
func (repo *GormRepository) GetFiles(q FilesQuery) (result []model.FileMeta, more bool, err error) {
	files := make([]*model.File, 0)
	tx := repo.db.Where("files.type = ?", q.Type.String())

	if q.ChannelID.Valid {
		if q.ChannelID.UUID == uuid.Nil {
			tx = tx.Where("files.channel_id IS NULL")
		} else {
			tx = tx.Where("files.channel_id = ?", q.ChannelID.UUID)
		}
	}
	if q.UploaderID.Valid {
		if q.UploaderID.UUID == uuid.Nil {
			tx = tx.Where("files.creator_id IS NULL")
		} else {
			tx = tx.Where("files.creator_id = ?", q.UploaderID.UUID)
		}
	}

	if q.Inclusive {
		if q.Since.Valid {
			tx = tx.Where("files.created_at >= ?", q.Since.Time)
		}
		if q.Until.Valid {
			tx = tx.Where("files.created_at <= ?", q.Until.Time)
		}
	} else {
		if q.Since.Valid {
			tx = tx.Where("files.created_at > ?", q.Since.Time)
		}
		if q.Until.Valid {
			tx = tx.Where("files.created_at < ?", q.Until.Time)
		}
	}

	if q.Asc {
		tx = tx.Order("files.created_at")
	} else {
		tx = tx.Order("files.created_at DESC")
	}

	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	if q.Limit > 0 {
		err = tx.Limit(q.Limit + 1).Find(&files).Error
		if len(files) > q.Limit {
			return repo.makeFileMetas(files[:len(files)-1]), true, err
		}
	} else {
		err = tx.Find(&files).Error
	}
	return repo.makeFileMetas(files), false, err
}

// SaveFile implements FileRepository interface.
func (repo *GormRepository) SaveFile(args SaveFileArgs) (model.FileMeta, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	f := &model.File{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      args.FileName,
		Size:      args.FileSize,
		Mime:      args.MimeType,
		Type:      args.FileType,
		CreatorID: args.CreatorID,
		ChannelID: args.ChannelID,
	}

	if args.Thumbnail != nil {
		f.HasThumbnail = true
		f.ThumbnailMime = null.StringFrom("image/png")
		f.ThumbnailWidth = args.Thumbnail.Bounds().Size().X
		f.ThumbnailHeight = args.Thumbnail.Bounds().Size().Y

		r, w := io.Pipe()
		go func() {
			defer w.Close()
			_ = png.Encode(w, args.Thumbnail)
		}()

		key := f.ID.String() + "-thumb"
		if err := repo.FS.SaveByKey(r, key, key+".png", "image/png", model.FileTypeThumbnail); err != nil {
			return nil, err
		}
	}

	hash := md5.New()
	if err := repo.FS.SaveByKey(io.TeeReader(args.Src, hash), f.ID.String(), f.Name, f.Mime, f.Type); err != nil {
		return nil, err
	}
	f.Hash = hex.EncodeToString(hash.Sum(nil))

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(f).Error; err != nil {
			return err
		}

		for uid, allow := range args.ACL {
			if err := tx.Create(&model.FileACLEntry{
				FileID: f.ID,
				UserID: uuid.NullUUID{UUID: uid, Valid: true},
				Allow:  sql.NullBool{Bool: allow, Valid: true},
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		_ = repo.FS.DeleteByKey(f.ID.String(), f.Type)
		if f.HasThumbnail {
			_ = repo.FS.DeleteByKey(f.ID.String()+"-thumb", model.FileTypeThumbnail)
		}
		return nil, err
	}
	return repo.makeFileMeta(f), nil
}

// GetFileMeta implements FileRepository interface.
func (repo *GormRepository) GetFileMeta(fileID uuid.UUID) (model.FileMeta, error) {
	if fileID == uuid.Nil {
		return nil, ErrNotFound
	}
	f := &model.File{}
	if err := repo.db.Take(f, &model.File{ID: fileID}).Error; err != nil {
		return nil, convertError(err)
	}
	return repo.makeFileMeta(f), nil
}

// DeleteFile implements FileRepository interface.
func (repo *GormRepository) DeleteFile(fileID uuid.UUID) error {
	if fileID == uuid.Nil {
		return ErrNilID
	}

	var f model.File
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Take(&f, &model.File{ID: fileID}).Error; err != nil {
			return convertError(err)
		}

		if err := tx.Delete(&f).Error; err != nil {
			return err
		}

		return repo.FS.DeleteByKey(f.ID.String(), f.Type)
	})
	if err != nil {
		return err
	}

	if f.HasThumbnail {
		// エラーを無視
		_ = repo.FS.DeleteByKey(f.ID.String()+"-thumb", model.FileTypeThumbnail)
	}
	return nil
}

// IsFileAccessible implements FileRepository interface.
func (repo *GormRepository) IsFileAccessible(fileID, userID uuid.UUID) (bool, error) {
	if fileID == uuid.Nil {
		return false, ErrNilID
	}

	if ok, err := dbExists(repo.db, &model.File{ID: fileID}); err != nil {
		return false, err
	} else if !ok {
		return false, ErrNotFound
	}

	var result struct {
		Allow int
		Deny  int
	}
	err := repo.db.
		Model(&model.FileACLEntry{}).
		Select("COUNT(allow = TRUE OR NULL) AS allow, COUNT(allow = FALSE OR NULL) AS deny").
		Where("file_id = ? AND user_id IN (?)", fileID, []uuid.UUID{userID, uuid.Nil}).
		Scan(&result).
		Error
	if err != nil {
		return false, err
	}
	return result.Allow > 0 && result.Deny == 0, nil
}

func (repo *GormRepository) makeFileMeta(f *model.File) model.FileMeta {
	return &fileMetaImpl{meta: f, fs: repo.FS}
}

func (repo *GormRepository) makeFileMetas(fs []*model.File) []model.FileMeta {
	result := make([]model.FileMeta, len(fs))
	for i, f := range fs {
		result[i] = repo.makeFileMeta(f)
	}
	return result
}
