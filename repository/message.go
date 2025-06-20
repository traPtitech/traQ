//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"time"

	"github.com/traPtitech/traQ/utils/optional"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// MessagesQuery GetMessages用クエリ
type MessagesQuery struct {
	IDIn    optional.Of[[]uuid.UUID]
	User    uuid.UUID
	Channel uuid.UUID
	// ChannelsSubscribedByUser 指定したユーザーが購読しているチャンネルのメッセージを指定
	ChannelsSubscribedByUser uuid.UUID
	Since                    optional.Of[time.Time]
	Until                    optional.Of[time.Time]
	Inclusive                bool
	Limit                    int
	Offset                   int
	Asc                      bool
	ExcludeDMs               bool
	DisablePreload           bool
}

// ChannelLatestMessagesQuery GetChannelLatestMessages用クエリ
type ChannelLatestMessagesQuery struct {
	// SubscribedByUser 指定したユーザーが購読しているチャンネル
	SubscribedByUser optional.Of[uuid.UUID]
	Limit            int
	Since            optional.Of[time.Time]
}

// MessageRepository メッセージリポジトリ
type MessageRepository interface {
	// CreateMessage メッセージを作成します
	//
	// 成功した場合、メッセージとnilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error)
	// UpdateMessage 指定したメッセージを更新します
	//
	// 成功した場合、nilを返します。
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
	// GetMessages 指定したクエリでメッセージを取得します
	//
	// 成功した場合、メッセージの配列を返します。負のoffset, limitは無視されます。
	// 指定した範囲内にlimitを超えてメッセージが存在していた場合、trueを返します。
	// DBによるエラーを返すことがあります。
	GetMessages(query MessagesQuery) (messages []*model.Message, more bool, err error)
	// GetUpdatedMessagesAfter 指定した時間より後に更新されたメッセージを取得します
	//
	// 成功した場合、updatedAtで昇順ソートされたメッセージの配列を返します。
	// 指定した範囲内にlimitを超えてメッセージが存在していた場合、trueを返します。
	// DBによるエラーを返すことがあります。
	GetUpdatedMessagesAfter(after time.Time, limit int) (messages []*model.Message, more bool, err error)
	// GetDeletedMessagesAfter 指定した時間より後に削除されたメッセージを取得します
	//
	// 成功した場合、deletedAtで昇順ソートされたメッセージの配列を返します。
	// 指定した範囲内にlimitを超えてメッセージが存在していた場合、trueを返します。
	// DBによるエラーを返すことがあります。
	GetDeletedMessagesAfter(after time.Time, limit int) (messages []*model.Message, more bool, err error)
	// SetMessageUnreads 指定した(複数の)ユーザーに対し、指定したメッセージを未読にします
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するもしくは引数中の要素にuuid.Nilが含まれるものを指定するととErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	SetMessageUnreads(userNoticeableMap map[uuid.UUID]bool, messageID uuid.UUID) error
	// GetUnreadMessagesByUserID 指定したユーザーの未読メッセージをすべて取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUnreadMessagesByUserID(userID uuid.UUID) ([]*model.Message, error)
	// DeleteUnreadsByChannelID 指定したチャンネルに存在する、指定したユーザーの未読レコードをすべて削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error
	// GetUserUnreadChannels 指定したユーザーの未読チャンネル一覧を取得します
	//
	// 成功した場合、UserUnreadChannelの配列とnilを返します。
	// 存在しないユーザーを指定した場合、空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserUnreadChannels(userID uuid.UUID) ([]*UserUnreadChannel, error)
	// GetChannelLatestMessages 全てのパブリックチャンネルの最新のメッセージの一覧を取得します
	//
	// 成功した場合、メッセージの配列とnilを返します。非正のlimitは無視されます。
	// DBによるエラーを返すことがあります。
	GetChannelLatestMessages(query ChannelLatestMessagesQuery) ([]*model.Message, error)
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
}

// UserUnreadChannel ユーザーの未読チャンネル構造体
type UserUnreadChannel struct {
	ChannelID       uuid.UUID `json:"channelId"`
	Count           int       `json:"count"`
	Noticeable      bool      `json:"noticeable"`
	Since           time.Time `json:"since"`
	UpdatedAt       time.Time `json:"updatedAt"`
	OldestMessageID uuid.UUID `json:"oldestMessageId"`
}
