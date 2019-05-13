package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// TagRepository ユーザータグリポジトリ
type TagRepository interface {
	// CreateTag タグを作成します
	//
	// 成功した場合、タグとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	CreateTag(name string, restricted bool, tagType string) (*model.Tag, error)
	// GetTagByID 指定したIDのタグを取得します
	//
	// 成功した場合、タグとnilを返します。
	// 存在しないタグの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetTagByID(id uuid.UUID) (*model.Tag, error)
	// GetTagByName 指定した名前のタグを取得します
	//
	// 成功した場合、タグとnilを返します。
	// 存在しないタグの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetTagByName(name string) (*model.Tag, error)
	// GetOrCreateTagByName 指定したタグを取得するか、生成したものを返します
	//
	// 成功した場合、タグとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 空文字を指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetOrCreateTagByName(name string) (*model.Tag, error)
	// AddUserTag 指定したユーザーに指定したタグを付与します
	//
	// 成功した場合、nilを返します。
	// 既に付与されている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddUserTag(userID, tagID uuid.UUID) error
	// ChangeUserTagLock 指定したユーザーの指定したタグのロック状態を変更します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error
	// DeleteUserTag 指定したユーザーから指定したタグを削除します
	//
	// 成功した、或いは既に無い場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteUserTag(userID, tagID uuid.UUID) error
	// GetUserTag 指定したユーザーの指定したタグのユーザータグを取得します
	//
	// 成功した場合、ユーザータグとnilを返します。
	// 存在しないユーザータグの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error)
	// GetUserTagsByUserID 指定したユーザーに付与されているタグを全て取得します
	//
	// 成功した場合、ユーザータグの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserTagsByUserID(userID uuid.UUID) ([]*model.UsersTag, error)
	// GetUserIDsByTag 指定したタグを持った全ユーザーのUUIDを取得します
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないタグを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserIDsByTag(tag string) ([]uuid.UUID, error)
	// GetUserIDsByTagID 指定したタグを持った全ユーザーのUUIDを取得します
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないタグを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error)
}
