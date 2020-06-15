package file

import (
	"errors"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
	"image"
	"io"
	"mime"
	"path/filepath"
)

var (
	ErrNotFound = errors.New("not found")
)

type SaveArgs struct {
	FileName                string
	FileSize                int64
	MimeType                string
	FileType                model.FileType
	CreatorID               optional.UUID
	ChannelID               optional.UUID
	ACL                     ACL
	Src                     io.Reader
	Thumbnail               image.Image
	SkipThumbnailGeneration bool
}

// ACL アクセスコントロールリスト
//
// keyとしてユーザーのUUIDを取り、valueとしてAllowをtrue、Denyをfalseで表します。
// keyとしてuuid.Nilを指定すると、全てのユーザーを表します。Denyルールが優先されます。
type ACL map[uuid.UUID]bool

func (args *SaveArgs) Validate() error {
	if len(args.MimeType) == 0 {
		args.MimeType = mime.TypeByExtension(filepath.Ext(args.FileName))
		if len(args.MimeType) == 0 {
			args.MimeType = "application/octet-stream"
		}
	}
	if args.ACL == nil {
		args.ACLAllow(uuid.Nil)
	}
	if args.CreatorID.Valid {
		args.ACLAllow(args.CreatorID.UUID)
	}
	return vd.ValidateStruct(args,
		vd.Field(&args.FileName, vd.Required),
		vd.Field(&args.FileSize, vd.Required, vd.Min(1)),
		vd.Field(&args.MimeType, vd.Required, is.PrintableASCII),
		vd.Field(&args.CreatorID, validator.NotNilUUID),
		vd.Field(&args.ChannelID, validator.NotNilUUID),
		vd.Field(&args.ACL, vd.Required),
		vd.Field(&args.Src, vd.NotNil),
	)
}

func (args *SaveArgs) ACLAllow(userID uuid.UUID) {
	if args.ACL == nil {
		args.ACL = ACL{}
	}
	args.ACL[userID] = true
}

type Manager interface {
	Save(args SaveArgs) (model.File, error)
	Get(id uuid.UUID) (model.File, error)
	List(q repository.FilesQuery) ([]model.File, bool, error)
	Delete(id uuid.UUID) error
	Accessible(fileID, userID uuid.UUID) (bool, error)
}
