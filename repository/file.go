//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// FilesQuery GetFiles用クエリ
type FilesQuery struct {
	UploaderID optional.Of[uuid.UUID]
	ChannelID  optional.Of[uuid.UUID]
	Since      optional.Of[time.Time]
	Until      optional.Of[time.Time]
	Inclusive  bool
	Limit      int
	Offset     int
	Asc        bool
	Type       model.FileType
}

// FileRepository ファイルリポジトリ
type FileRepository interface {
	// GetFileMetas 指定したクエリでファイル情報一覧を取得します
	//
	// 成功した場合、ファイル情報の配列を返します。正でないoffset, limitは無視されます。
	// 指定した範囲内にlimitを超えてファイルが存在していた場合、trueを返します。
	// DBによるエラーを返すことがあります。
	GetFileMetas(q FilesQuery) (result []*model.FileMeta, more bool, err error)
	// GetFileMeta 指定したファイル情報を取得します
	//
	// 成功した場合、ファイル情報とnilを返します。
	// 存在しないファイルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetFileMeta(fileID uuid.UUID) (*model.FileMeta, error)
	// SaveFileMeta ファイル情報と、metaに含まれるサムネイル情報を格納します
	//
	// 成功した場合、nilを返します。
	// metaに指定されたIDがnilの場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	SaveFileMeta(meta *model.FileMeta, acl []*model.FileACLEntry) error
	// DeleteFileMeta ファイル情報を削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteFileMeta(fileID uuid.UUID) error
	// IsFileAccessible ユーザーがファイルへのアクセス権限を持っているかを確認します
	//
	// ユーザーがアクセス権限を持っている場合、trueを返します。
	// ファイルもしくはユーザーが存在しない場合は、falseを返します。
	// DBによるエラーを返すことがあります。
	IsFileAccessible(fileID, userID uuid.UUID) (bool, error)
	// DeleteFileThumbnail サムネイル情報を削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteFileThumbnail(fileId uuid.UUID, thumbnailType model.ThumbnailType) error
}
