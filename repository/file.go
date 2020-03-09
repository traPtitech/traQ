package repository

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
	"io"
	"mime"
	"path/filepath"
)

type SaveFileArgs struct {
	FileName  string
	FileSize  int64
	MimeType  string
	FileType  string
	CreatorID uuid.NullUUID
	ChannelID uuid.NullUUID
	ACL       ACL
	Src       io.Reader
}

func (args *SaveFileArgs) Validate() error {
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

func (args *SaveFileArgs) SetChannel(id uuid.UUID) {
	args.ChannelID = uuid.NullUUID{Valid: true, UUID: id}
}

func (args *SaveFileArgs) SetCreator(id uuid.UUID) {
	args.CreatorID = uuid.NullUUID{Valid: true, UUID: id}
}

func (args *SaveFileArgs) ACLAllow(userID uuid.UUID) {
	if args.ACL == nil {
		args.ACL = ACL{}
	}
	args.ACL[userID] = true
}

// FileRepository ファイルリポジトリ
type FileRepository interface {
	// OpenFile 指定したファイルのストリームを開きます
	//
	// 成功した場合、メタデータとストリームとnilを返します。
	// 存在しないファイルを指定した場合、ErrNotFoundを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	OpenFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error)
	// OpenThumbnailFile 指定したファイルのサムネイルのストリームを開きます
	//
	// 成功した場合、メタデータとストリームとnilを返します。
	// 存在しないファイル、或いはサムネイルが存在しないファイルを指定した場合、ErrNotFoundを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	OpenThumbnailFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error)
	// GetFileMeta 指定したファイルのメタデータを取得します
	//
	// 成功した場合、メタデータとnilを返します。
	// 存在しないファイルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetFileMeta(fileID uuid.UUID) (*model.File, error)
	// DeleteFile 指定したファイルを削除します
	//
	// 成功した場合、nilを返します。ファイルデータは完全に削除されます。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないファイルを指定した場合、ErrNotFoundを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	DeleteFile(fileID uuid.UUID) error
	// GenerateIconFile アイコンファイルを生成します
	//
	// 成功した場合、そのファイルのUUIDとnilを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	GenerateIconFile(salt string) (uuid.UUID, error)
	// SaveFile ファイルを保存します
	//
	// mimeが指定されていない場合はnameの拡張子によって決まります。
	// 成功した場合、メタデータとnilを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	SaveFile(args SaveFileArgs) (*model.File, error)
	// IsFileAccessible 指定したユーザーが指定したファイルにアクセス可能かどうかを返します
	//
	// アクセス可能な場合、trueとnilを返します。
	// fileIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないfileIDを指定した場合、ErrNotFoundを返します。
	// userIDにuuid.Nilを指定すると、全てのユーザーを指定します。全てのユーザーに関するACLの設定を返します。全てのユーザーがアクセス可能な場合にのみtrueを返すとは限りません。
	// DBによるエラーを返すことがあります。
	IsFileAccessible(fileID, userID uuid.UUID) (bool, error)
}

// ACL アクセスコントロールリスト
//
// keyとしてユーザーのUUIDを取り、valueとしてAllowをtrue、Denyをfalseで表します。
// keyとしてuuid.Nilを指定すると、全てのユーザーを表します。Denyルールが優先されます。
type ACL map[uuid.UUID]bool
