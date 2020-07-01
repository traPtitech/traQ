package file

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/ioext"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/storage"
	"time"
)

type fileMetaImpl struct {
	meta *model.FileMeta
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

func (f *fileMetaImpl) GetCreatorID() optional.UUID {
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

func (f *fileMetaImpl) GetUploadChannelID() optional.UUID {
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
		return nil, fmt.Errorf("no thumbnail image")
	}
	return f.fs.OpenFileByKey(f.GetID().String()+"-thumb", model.FileTypeThumbnail)
}

func (f *fileMetaImpl) GetAlternativeURL() string {
	url, _ := f.fs.GenerateAccessURL(f.GetID().String(), f.GetFileType())
	return url
}
