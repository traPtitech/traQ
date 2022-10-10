//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// CreateUserArgs ユーザー作成引数
type CreateUserArgs struct {
	Name          string
	DisplayName   string
	Role          string
	IconFileID    uuid.UUID
	Password      string
	ExternalLogin *model.ExternalProviderUser
}

// UpdateUserArgs User情報更新引数
type UpdateUserArgs struct {
	DisplayName optional.Of[string]
	TwitterID   optional.Of[string]
	Role        optional.Of[string]
	UserState   optional.Of[model.UserAccountStatus]
	Bio         optional.Of[string]
	IconFileID  optional.Of[uuid.UUID]
	LastOnline  optional.Of[time.Time]
	HomeChannel optional.Of[uuid.UUID]
	Password    optional.Of[string]
}

// LinkExternalUserAccountArgs 外部アカウント関連付け引数
type LinkExternalUserAccountArgs struct {
	ProviderName string
	ExternalID   string
	Extra        model.JSON
}

// UsersQuery GetUsers用クエリ
type UsersQuery struct {
	Name                        optional.Of[string]
	IsBot                       optional.Of[bool]
	IsActive                    optional.Of[bool]
	IsCMemberOf                 optional.Of[uuid.UUID]
	IsGMemberOf                 optional.Of[uuid.UUID]
	IsSubscriberAtMarkLevelOf   optional.Of[uuid.UUID]
	IsSubscriberAtNotifyLevelOf optional.Of[uuid.UUID]
	EnableProfileLoading        bool
}

// NotBot Botでない
func (q UsersQuery) NotBot() UsersQuery {
	q.IsBot = optional.From(false)
	return q
}

// NameOf nameの名前のユーザーである
func (q UsersQuery) NameOf(name string) UsersQuery {
	q.Name = optional.From(name)
	return q
}

// Active アカウントが有効である
func (q UsersQuery) Active() UsersQuery {
	q.IsActive = optional.From(true)
	return q
}

// CMemberOf channelIDプライベートチャンネルのメンバーである
func (q UsersQuery) CMemberOf(channelID uuid.UUID) UsersQuery {
	q.IsCMemberOf = optional.From(channelID)
	return q
}

// GMemberOf groupIDグループのメンバーである
func (q UsersQuery) GMemberOf(groupID uuid.UUID) UsersQuery {
	q.IsGMemberOf = optional.From(groupID)
	return q
}

// SubscriberAtMarkLevelOf channelIDチャンネルの未読管理レベル購読ユーザーである
func (q UsersQuery) SubscriberAtMarkLevelOf(channelID uuid.UUID) UsersQuery {
	q.IsSubscriberAtMarkLevelOf = optional.From(channelID)
	return q
}

// SubscriberAtNotifyLevelOf channelIDチャンネルの通知レベル購読ユーザーである
func (q UsersQuery) SubscriberAtNotifyLevelOf(channelID uuid.UUID) UsersQuery {
	q.IsSubscriberAtNotifyLevelOf = optional.From(channelID)
	return q
}

// LoadProfile ユーザーの追加プロファイル情報を読み込むかどうか
func (q UsersQuery) LoadProfile() UsersQuery {
	q.EnableProfileLoading = true
	return q
}

// UserStats ユーザー統計情報
type UserStats struct {
	TotalMessageCount int64 `json:"totalMessageCount"`
	Stamps            []struct {
		ID    uuid.UUID `json:"id"`
		Count int64     `json:"count"`
		Total int64     `json:"total"`
	} `json:"stamps"`
	DateTime time.Time `json:"datetime"`
}

// UserRepository ユーザーリポジトリ
type UserRepository interface {
	// CreateUser ユーザーを作成します
	//
	// 成功した場合、ユーザーとnilを返します。
	// Nameが既に使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateUser(args CreateUserArgs) (model.UserInfo, error)
	// GetUser 指定したIDのユーザーを取得します
	//
	// 成功した場合、ユーザーとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUser(id uuid.UUID, withProfile bool) (model.UserInfo, error)
	// GetUserByName 指定したNameのユーザーを取得する
	//
	// 成功した場合、ユーザーとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserByName(name string, withProfile bool) (model.UserInfo, error)
	// GetUserByExternalID 指定したproviderのexternalIDのユーザーを取得する
	//
	// 成功した場合、ユーザーとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserByExternalID(providerName, externalID string, withProfile bool) (model.UserInfo, error)
	// GetUsers 指定した条件を満たすユーザーを取得します
	//
	// 成功した場合、ユーザーの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUsers(query UsersQuery) ([]model.UserInfo, error)
	// GetUserIDs 指定した条件を満たすユーザーのUUIDの配列を取得します
	//
	// 成功した場合、UUIDの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserIDs(query UsersQuery) ([]uuid.UUID, error)
	// UserExists 指定したIDのユーザーが存在するかどうかを返します
	//
	// 存在する場合、trueとnilを返します。
	// DBによるエラーを返すことがあります。
	UserExists(id uuid.UUID) (bool, error)
	// UpdateUser 指定したユーザーの情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないユーザーの場合、ErrNotFoundを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateUser(id uuid.UUID, args UpdateUserArgs) error
	// LinkExternalUserAccount 指定したユーザーに外部ログインアカウントを関連付けします
	//
	// 成功した場合、nilを返します。
	// 存在しないユーザーの場合、ErrNotFoundを返します。
	// 既に指定された外部プロバイダとの関連付けがある場合、ErrAlreadyExistを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	LinkExternalUserAccount(userID uuid.UUID, args LinkExternalUserAccountArgs) error
	// GetLinkedExternalUserAccounts 指定したユーザーに関連づけられている外部ログインアカウントの配列を返します
	//
	// 成功した場合、外部ログインアカウントの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetLinkedExternalUserAccounts(userID uuid.UUID) ([]*model.ExternalProviderUser, error)
	// UnlinkExternalUserAccount 指定したユーザーに関連づけられている指定した外部ログインアカウントの関連付けを解除します
	//
	// 成功した場合、nilを返します。
	// 既に関連付けが無い場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UnlinkExternalUserAccount(userID uuid.UUID, providerName string) error
	// GetUserStats 成功した場合、(統計情報, nil)を返します。
	//
	// ユーザーが存在しない場合、(nil, ErrNotFound)を返します。
	// 引数にuuid.Nilを指定した場合、(nil, ErrNilID)を返します。
	// DBによるエラーを返すことがあります。
	GetUserStats(userID uuid.UUID) (*UserStats, error)
}
