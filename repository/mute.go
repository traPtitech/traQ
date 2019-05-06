package repository

import "github.com/gofrs/uuid"

// MuteRepository ミュートリポジトリ
type MuteRepository interface {
	// MuteChannel 指定したチャンネルをミュートします
	//
	// 成功した、或いは既に登録されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	MuteChannel(userID, channelID uuid.UUID) error
	// UnmuteChannel 指定したチャンネルのミュートを解除します
	//
	// 成功した、或いは既にミュート解除されていた場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UnmuteChannel(userID, channelID uuid.UUID) error
	// GetMutedChannelIDs ミュートしているチャンネルのIDの配列を取得します
	//
	// 成功した場合、チャンネルUUIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMutedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error)
	// GetMuteUserIDs ミュートしているユーザーのIDの配列を取得します
	//
	// 成功した場合、ユーザーUUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMuteUserIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// IsChannelMuted 指定したユーザーが指定したチャンネルをミュートしているかどうかを返します
	//
	// ミュートしている場合、trueとnilを返します。
	// 存在しないユーザーやチャンネルを指定した場合はfalseとnilを返します。
	// DBによるエラーを返すことがあります。
	IsChannelMuted(userID, channelID uuid.UUID) (bool, error)
}
