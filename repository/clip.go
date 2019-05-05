package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// ClipRepository クリップリポジトリ
type ClipRepository interface {
	// GetClipFolder 指定したクリップフォルダを取得します
	//
	// 成功した場合、クリップフォルダとnilを返します。
	// 存在しないフォルダを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetClipFolder(id uuid.UUID) (*model.ClipFolder, error)
	// GetClipFolders 指定したユーザーのクリップフォルダを全て取得します
	//
	// 成功した場合、クリップフォルダの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetClipFolders(userID uuid.UUID) ([]*model.ClipFolder, error)
	// CreateClipFolder クリップフォルダを作成します
	//
	// 成功した場合、フォルダとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既に使われている名前の場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error)
	// UpdateClipFolderName 指定したクリップフォルダの名前を変更します
	//
	// 成功した場合、nilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既に使われている名前の場合、ErrAlreadyExistsを返します。
	// 存在しないフォルダを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateClipFolderName(id uuid.UUID, name string) error
	// DeleteClipFolder 指定したクリップフォルダを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しないフォルダを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteClipFolder(id uuid.UUID) error
	// GetClipMessage 指定したクリップを取得します
	//
	// 成功した場合、クリップとnilを返します。
	// 存在しないクリップを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetClipMessage(id uuid.UUID) (*model.Clip, error)
	// GetClipMessages 指定したフォルダのクリップを全て取得します
	//
	// 成功した場合、クリップの配列とnilを返します。
	// 存在しないフォルダを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetClipMessages(folderID uuid.UUID) ([]*model.Clip, error)
	// GetClipMessagesByUser 指定したユーザーのクリップを全て取得します
	//
	// 成功した場合、クリップの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetClipMessagesByUser(userID uuid.UUID) ([]*model.Clip, error)
	// CreateClip クリップを作成します
	//
	// 成功した場合、クリップとnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateClip(messageID, folderID, userID uuid.UUID) (*model.Clip, error)
	// ChangeClipFolder 指定したクリップのフォルダを変更します
	//
	// 成功した場合、nilを返します。
	// 存在しないクリップを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeClipFolder(clipID, folderID uuid.UUID) error
	// DeleteClip 指定したクリップを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しないクリップを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteClip(id uuid.UUID) error
}
