package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageStampRepository メッセージスタンプリポジトリ
type MessageStampRepository interface {
	// AddStampToMessage 指定したメッセージに指定したユーザーの指定したスタンプを追加します
	//
	// 成功した場合、そのメッセージスタンプとnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	AddStampToMessage(messageID, stampID, userID uuid.UUID, count int) (ms *model.MessageStamp, err error)
	// RemoveStampFromMessage 指定したメッセージから指定したユーザーの指定したスタンプを全て削除します
	//
	// 成功した、或いは既に削除されていた場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error)
	// GetMessageStamps 指定したメッセージのスタンプを全て取得します
	//
	// 成功した場合、メッセージスタンプの配列とnilを返します。
	// 存在しないメッセージを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error)
	// GetUserStampHistory 指定したユーザーのスタンプ履歴を最大100件取得します
	//
	// 成功した場合、降順のスタンプ履歴の配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error)
}
