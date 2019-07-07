package repository

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

var (
	// ErrChannelDepthLimitation チャンネルの深さ制限を超えている
	ErrChannelDepthLimitation = errors.New("channel depth limit exceeded")
)

// ChangeChannelSubscriptionArgs チャンネル購読変更引数
type ChangeChannelSubscriptionArgs struct {
	UpdaterID    uuid.UUID
	Subscription map[uuid.UUID]bool
}

// ChannelRepository チャンネルリポジトリ
type ChannelRepository interface {
	// CreatePublicChannel パブリックチャンネルを作成します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 作成不可能な親チャンネルを指定した場合、ErrForbiddenを返します。
	// 階層数制限に到達する場合、ErrChannelDepthLimitationを返します。
	// 存在しない親チャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error)
	// CreatePrivateChannel プライベートチャンネルを作成します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error)
	// CreateChildChannel 子チャンネルを作成します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 作成不可能な親チャンネルを指定した場合、ErrForbiddenを返します。
	// 存在しない親チャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error)
	// UpdateChannelAttributes 指定したチャンネルの属性を変更します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error
	// UpdateChannelTopic 指定したチャンネルのトピックを更新します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error
	// ChangeChannelName 指定したチャンネルのチャンネル名を変更します
	//
	// 成功した場合、nilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 変更不可能なチャンネルを指定した場合、ErrForbiddenを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	ChangeChannelName(channelID uuid.UUID, name string) error
	// ChangeChannelParent 指定したチャンネルの親を変更します
	//
	// 成功した場合、nilを返します。
	// Nameが変更先で重複している場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 変更不可能なチャンネルを指定した場合、ErrForbiddenを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// 階層数制限に到達する場合、ErrChannelDepthLimitationを返します。
	// DBによるエラーを返すことがあります。
	ChangeChannelParent(channelID, parent uuid.UUID) error
	// DeleteChannel 指定したチャンネルとその子孫チャンネルを全て削除します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	DeleteChannel(channelID uuid.UUID) error
	// GetChannel 指定したチャンネルを取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannel(channelID uuid.UUID) (*model.Channel, error)
	// GetChannelByMessageID 指定したメッセージの投稿先チャンネルを取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 存在しないメッセージを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error)
	// GetChannelsByUserID 指定したユーザーがアクセス可能なチャンネルを全て取得します
	//
	// 成功した場合、チャンネルの配列とnilを返します。
	// 存在しないユーザーやuuid.Nilを指定した場合、パブリックチャンネルのみを返します。
	// DBによるエラーを返すことがあります。
	GetChannelsByUserID(userID uuid.UUID) ([]*model.Channel, error)
	// GetDirectMessageChannel 引数に指定したユーザー間のDMチャンネル取得します
	//
	// 成功した場合、チャンネルとnilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error)
	// IsChannelAccessibleToUser 指定したチャンネルが指定したユーザーからアクセス可能かどうかを返します
	//
	// アクセス可能な場合、trueとnilを返します。
	// 存在しないチャンネル・ユーザーを指定した場合、falseとnilを返します。
	// DBによるエラーを返すことがあります。
	IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error)
	// GetChildrenChannelIDs 指定したチャンネルの子チャンネルのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetChildrenChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// GetChannelPath 指定したチャンネルのパス文字列を取得する
	//
	// 成功した場合、パス文字列とnilを返します。
	// 存在しないチャンネルを指定した場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetChannelPath(id uuid.UUID) (string, error)
	// GetPrivateChannelMemberIDs 指定したプライベートチャンネルのメンバーのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// ChangeChannelSubscription ユーザーのチャンネルの購読を変更します
	//
	// 成功した場合、nilを返します。
	// channelIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 存在しないユーザーを指定した場合は無視されます。
	// DBによるエラーを返すことがあります。
	ChangeChannelSubscription(channelID uuid.UUID, args ChangeChannelSubscriptionArgs) error
	// GetSubscribingUserIDs 指定したチャンネルを購読しているユーザーのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないチャンネルを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetSubscribingUserIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	// GetSubscribedChannelIDs 指定したユーザーが購読しているチャンネルのUUIDを全て取得する
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetSubscribedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error)
}
