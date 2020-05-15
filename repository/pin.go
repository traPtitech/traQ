package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// PinRepository ピンリポジトリ
type PinRepository interface {
	// PinMessage 指定したユーザーによって指定したメッセージをピン留めします
	//
	// 成功した、或いは既にピン留めされていた場合、ピン留めのUUIDとnilを返します。既にピン留めされていた場合にユーザーIDは上書きされません。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// 存在しないメッセージを指定した場合はErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	PinMessage(messageID, userID uuid.UUID) (*model.Pin, error)
	// UnpinMessage 指定したユーザーによって指定したピン留めを削除します
	//
	// 成功した、或いは既にピン留めされていなかった場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UnpinMessage(messageID, userID uuid.UUID) error
	// GetPinnedMessageByChannelID 指定したチャンネルのピン留めを全て取得します
	//
	// 成功した場合、ピン留めの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetPinnedMessageByChannelID(channelID uuid.UUID) ([]*model.Pin, error)
}
