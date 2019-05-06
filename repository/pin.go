package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// PinRepository ピンリポジトリ
type PinRepository interface {
	// CreatePin 指定したユーザーによって指定したメッセージをピン留めします
	//
	// 成功した、或いは既にピン留めされていた場合、ピン留めのUUIDとnilを返します。既にピン留めされていた場合にユーザーIDは上書きされません。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error)
	// GetPin 指定したピン留めを取得します
	//
	// 成功した場合、ピン留めとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetPin(id uuid.UUID) (*model.Pin, error)
	// IsPinned 指定したメッセージがピン留めされているかどうかを返します
	//
	// ピン留めされている場合、trueとnilを返します。
	// 存在しないメッセージを指定した場合はfalseとnilを返します。
	// DBによるエラーを返すことがあります。
	IsPinned(messageID uuid.UUID) (bool, error)
	// DeletePin 指定したピン留めを削除します
	//
	// 成功した、或いは既にピン留めされていなかった場合にnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeletePin(id uuid.UUID) error
	// GetPinsByChannelID 指定したチャンネルのピン留めを全て取得します
	//
	// 成功した場合、ピン留めの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetPinsByChannelID(channelID uuid.UUID) ([]*model.Pin, error)
}
