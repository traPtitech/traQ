package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageRepository メッセージリポジトリ
type MessageRepository interface {
	// CreateMessage メッセージを作成します
	//
	// 成功した場合、メッセージとnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// textが空の場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error)
	// UpdateMessage 指定したメッセージを更新します
	//
	// 成功した場合、nilを返します。
	// textが空の場合、ArgumentErrorを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateMessage(messageID uuid.UUID, text string) error
	// DeleteMessage 指定したメッセージを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteMessage(messageID uuid.UUID) error
	// GetMessageByID 指定したメッセージを取得します
	//
	// 成功した場合、メッセージとnilを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetMessageByID(messageID uuid.UUID) (*model.Message, error)
	// GetMessagesByChannelID 指定したチャンネルのメッセージを取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。負のoffset, limitは無視されます。
	// 存在しないチャンネルを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) ([]*model.Message, error)
	// GetMessagesByUserID 指定したユーザーのメッセージを取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。負のoffset, limitは無視されます。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMessagesByUserID(userID uuid.UUID, limit, offset int) ([]*model.Message, error)
	// SetMessageUnread 指定したメッセージを未読にします
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	SetMessageUnread(userID, messageID uuid.UUID) error
	// GetUnreadMessagesByUserID 指定したユーザーの未読メッセージをすべて取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUnreadMessagesByUserID(userID uuid.UUID) ([]*model.Message, error)
	// DeleteUnreadsByMessageID 指定したメッセージの未読レコードを全て削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteUnreadsByMessageID(messageID uuid.UUID) error
	// DeleteUnreadsByChannelID 指定したチャンネルに存在する、指定したユーザーの未読レコードをすべて削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error
	// GetChannelLatestMessagesByUserID 指定したユーザーが閲覧可能な全てのパブリックチャンネルの最新のメッセージの一覧を取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。負のlimitは無視されます。
	// DBによるエラーを返すことがあります。
	GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error)
	// GetArchivedMessagesByID 指定したメッセージのアーカイブメッセージを取得します
	//
	// 成功した場合、アーカイブメッセージの配列とnilを返します。
	// 存在しないメッセージを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetArchivedMessagesByID(messageID uuid.UUID) ([]*model.ArchivedMessage, error)
}
