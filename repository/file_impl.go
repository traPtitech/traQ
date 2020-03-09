package repository

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gopkg.in/guregu/null.v3"
	"image"
	"io"
)

type fileImpl struct {
	FS storage.FileStorage
}

// GenerateIconFile implements FileRepository interface.
func (repo *GormRepository) GenerateIconFile(salt string) (uuid.UUID, error) {
	var img bytes.Buffer
	_ = imaging.Encode(&img, utils.GenerateIcon(salt), imaging.PNG)
	file, err := repo.SaveFile(SaveFileArgs{
		FileName: fmt.Sprintf("%s.png", salt),
		FileSize: int64(img.Len()),
		MimeType: "image/png",
		FileType: model.FileTypeIcon,
		Src:      &img,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return file.ID, nil
}

// SaveFile implements FileRepository interface.
func (repo *GormRepository) SaveFile(args SaveFileArgs) (*model.File, error) {
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

	eg, ctx := errgroup.WithContext(context.Background())

	fileSrc, fileWriter := io.Pipe()
	thumbSrc, thumbWriter := io.Pipe()
	hash := md5.New()

	go func() {
		defer fileWriter.Close()
		defer thumbWriter.Close()
		_, _ = io.Copy(utils.MultiWriter(fileWriter, hash, thumbWriter), args.Src) // 並列化してるけど、pipeじゃなくてbuffer使わないとpipeがブロックしてて意味無い疑惑
	}()

	// fileの保存
	eg.Go(func() error {
		defer fileSrc.Close()
		return repo.FS.SaveByKey(fileSrc, f.GetKey(), f.Name, f.Mime, f.Type)
	})

	// サムネイルの生成
	eg.Go(func() error {
		// アップロードされたファイルの拡張子が間違えてたり、変なの送ってきた場合
		// サムネイルを生成しないだけで全体のエラーにはしない
		defer thumbSrc.Close()
		size, _ := repo.generateThumbnail(ctx, f, thumbSrc)
		if !size.Empty() {
			f.HasThumbnail = true
			f.ThumbnailMime = null.StringFrom("image/png")
			f.ThumbnailWidth = size.Size().X
			f.ThumbnailHeight = size.Size().Y
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
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
		return nil, err
	}
	return f, nil
}

// OpenFile implements FileRepository interface.
func (repo *GormRepository) OpenFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := repo.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	r, err := repo.FS.OpenFileByKey(meta.GetKey(), meta.Type)
	return meta, r, err
}

// OpenThumbnailFile implements FileRepository interface.
func (repo *GormRepository) OpenThumbnailFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := repo.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	if meta.HasThumbnail {
		r, err := repo.FS.OpenFileByKey(meta.GetThumbKey(), model.FileTypeThumbnail)
		return meta, r, err
	}
	return meta, nil, ErrNotFound
}

// GetFileMeta implements FileRepository interface.
func (repo *GormRepository) GetFileMeta(fileID uuid.UUID) (*model.File, error) {
	if fileID == uuid.Nil {
		return nil, ErrNotFound
	}
	f := &model.File{}
	if err := repo.db.Take(f, &model.File{ID: fileID}).Error; err != nil {
		return nil, convertError(err)
	}
	return f, nil
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

		return repo.FS.DeleteByKey(f.GetKey(), f.Type)
	})
	if err != nil {
		return err
	}

	if f.HasThumbnail {
		// エラーを無視
		_ = repo.FS.DeleteByKey(f.GetThumbKey(), model.FileTypeThumbnail)
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

var generateThumbnailS = semaphore.NewWeighted(5) // サムネイル生成並列数

// generateThumbnail サムネイル画像を生成します
func (repo *GormRepository) generateThumbnail(ctx context.Context, f *model.File, src io.Reader) (image.Rectangle, error) {
	if err := generateThumbnailS.Acquire(ctx, 1); err != nil {
		return image.Rectangle{}, err
	}
	defer generateThumbnailS.Release(1)

	orig, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return image.Rectangle{}, err
	}

	img := imaging.Fit(orig, 360, 480, imaging.Linear)

	r, w := io.Pipe()
	go func() {
		_ = imaging.Encode(w, img, imaging.PNG)
		_ = w.Close()
	}()

	if err := repo.FS.SaveByKey(r, f.GetThumbKey(), f.GetThumbKey()+".png", "image/png", model.FileTypeThumbnail); err != nil {
		return image.Rectangle{}, err
	}
	return img.Bounds(), nil
}
