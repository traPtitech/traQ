package message

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrChannelArchived  = errors.New("channel archived")
	ErrPinLimitExceeded = errors.New("the pin limit exceeded")
)

type TimelineQuery struct {
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

type Manager interface {
	// Get 指定したIDのメッセージを取得します
	//
	// 成功した場合、メッセージとnilを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	Get(id uuid.UUID) (Message, error)
	// GetIn 指定したIDのメッセージをすべて取得します
	// 存在チェックはせず、存在するメッセージだけを返します。
	// キャッシュをバイパスすることに気をつけてください。
	//
	// 成功した場合、メッセージとnilを返します。
	// DBによるエラーを返すことがあります。
	GetIn(ids []uuid.UUID) ([]Message, error)
	// GetTimeline タイムラインを取得します
	//
	// 成功した場合、タイムラインとnilを返します。
	// DBによるエラーを返すことがあります。
	GetTimeline(query TimelineQuery) (Timeline, error)
	// Create メッセージを作成します
	//
	// 成功した場合、メッセージとnilを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// DBによるエラーを返すことがあります。
	Create(channelID, userID uuid.UUID, content string) (Message, error)
	// CreateDM ダイレクトメッセージを作成します
	//
	// 成功した場合、メッセージとnilを返します。
	// DBによるエラーを返すことがあります。
	CreateDM(from, to uuid.UUID, content string) (Message, error)
	// Edit 指定したメッセージを編集します
	//
	// 成功した場合、nilを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	Edit(id uuid.UUID, content string) error
	// Delete 指定したメッセージを削除します
	//
	// 成功した場合、nilを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	Delete(id uuid.UUID) error
	// Pin 指定したユーザーによって指定したメッセージをピン留めします
	//
	// 成功した場合は、ピンとnilを返します。
	// 既にピンされている場合は、ErrAlreadyExistsを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合は、ErrNotFoundを返します。
	// チャンネルに既に上限数以上のメッセージがピン留めされていた場合、ErrPinLimitExceededを返します。
	// DBによるエラーを返すことがあります。
	Pin(id uuid.UUID, userID uuid.UUID) (*model.Pin, error)
	// Unpin 指定したユーザーによって指定したメッセージのピン留めを外します
	//
	// 成功した場合は、nilを返します。
	// 既にピンが無い場合は、ErrNotFoundを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合は、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	Unpin(id uuid.UUID, userID uuid.UUID) error
	// AddStamps 指定したメッセージに指定したユーザーの指定したスタンプを追加します
	//
	// 成功した場合、そのメッセージスタンプとnilを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合は、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	AddStamps(id, stampID, userID uuid.UUID, n int) (*model.MessageStamp, error)
	// RemoveStamps 指定したメッセージから指定したユーザーの指定したスタンプを全て削除します
	//
	// 成功した、或いは既に削除されていた場合、nilを返します。
	// アーカイブされているチャンネルを指定すると、ErrChannelArchivedを返します。
	// 存在しないメッセージを指定した場合は、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	RemoveStamps(id, stampID, userID uuid.UUID) error
	// IsAccessible 指定したユーザーが指定したメッセージにアクセス可能かどうかを確認します
	//
	// 成功した場合、アクセス可能かどうかとnilを返します。
	// 存在しないメッセージを指定した場合、falseとErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	IsAccessible(message Message, userID uuid.UUID) (bool, error)

	Wait(ctx context.Context) error
}
