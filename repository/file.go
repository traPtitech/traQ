package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"io"
)

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
	// SaveFileWithACLの引数creatorIDにuuid.NullUUID、readにACL{uuid.Nil: true}を指定したものと同じです。
	SaveFile(name string, src io.Reader, size int64, mime string, fType string) (*model.File, error)
	// SaveFileWithACL ファイルを保存します
	//
	// mimeが指定されていない場合はnameの拡張子によって決まります。
	// 成功した場合、メタデータとnilを返します。
	// DB, ファイルシステムによるエラーを返すことがあります。
	SaveFileWithACL(name string, src io.Reader, size int64, mime string, fType string, creatorID uuid.NullUUID, read ACL) (*model.File, error)
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
