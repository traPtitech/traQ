package file

import (
	"errors"
	"image"
	"io"
	"mime"
	"path/filepath"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
)

var (
	ErrNotFound = errors.New("not found")
)

type SaveArgs struct {
	FileName  string
	FileSize  int64
	MimeType  string
	FileType  model.FileType
	CreatorID optional.UUID
	ChannelID optional.UUID
	ACL       ACL
	Src       io.Reader
	Thumbnail image.Image
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
	// Save ファイルを保存します
	// サムネイルが生成可能な場合はサムネイルを生成し同時に保存します
	//
	// 成功した場合、ファイルとnilを返します。
	Save(args SaveArgs) (model.File, error)
	// Get ファイルを取得します
	//
	// 成功した場合、ファイルとnilを返します。
	Get(id uuid.UUID) (model.File, error)
	// List ファイルの一覧を取得します
	//
	// 成功した場合、ファイルの一覧を返します。負のoffset, limitは無視されます。
	// 指定した範囲内にlimitを超えてメッセージが存在していた場合、trueを返します。
	List(q repository.FilesQuery) ([]model.File, bool, error)
	// Delete ファイルを削除します
	//
	// 成功した場合、nilを返します。
	Delete(id uuid.UUID) error
	// Accessible ユーザーがファイルへのアクセス権限を持っているかを確認します
	//
	// ユーザーがアクセス権限を持っている場合、trueを返します。
	// ファイルもしくはユーザーが存在しない場合は、falseを返します。
	Accessible(fileID, userID uuid.UUID) (bool, error)
}
