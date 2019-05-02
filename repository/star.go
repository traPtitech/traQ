package repository

import "github.com/gofrs/uuid"

// StarRepository チャンネルスターリポジトリ
type StarRepository interface {
	// AddStar チャンネルをお気に入り登録します
	//
	// 成功した、或いは既に登録されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラー返すことがあります。
	AddStar(userID, channelID uuid.UUID) error
	// RemoveStar チャンネルのお気に入りを解除します
	//
	// 成功した、或いは既に解除されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラー返すことがあります。
	RemoveStar(userID, channelID uuid.UUID) error
	// GetStaredChannels ユーザーがお気に入りをしているチャンネルIDを取得します
	//
	// 成功した場合、チャンネルUUIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラー返すことがあります。
	GetStaredChannels(userID uuid.UUID) ([]uuid.UUID, error)
}
