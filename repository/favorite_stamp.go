package repository

import (
	"github.com/gofrs/uuid"
)

// FavoriteStampRepository お気に入りスタンプリポジトリ
type FavoriteStampRepository interface {
	// AddFavoriteStamp 指定したスタンプを指定したユーザーのお気に入りスタンプに追加します
	//
	// 成功した、或いは既に登録されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// 存在しないスタンプを指定した場合はArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	AddFavoriteStamp(userID, stampID uuid.UUID) error
	// RemoveFavoriteStamp 指定したスタンプを指定したユーザーのお気に入りスタンプから削除します
	//
	// 成功した、或いは既に解除されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveFavoriteStamp(userID, stampID uuid.UUID) error
	// GetUserFavoriteStamps 指定したユーザーのお気に入りスタンプのUUIDの配列を取得します
	//
	// 成功した場合、スタンプUUIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserFavoriteStamps(userID uuid.UUID) ([]uuid.UUID, error)
}
