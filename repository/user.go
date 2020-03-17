package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateUserArgs User情報更新引数
type UpdateUserArgs struct {
	DisplayName null.String
	TwitterID   null.String
	Role        null.String
	UserState   struct {
		Valid bool
		State model.UserAccountStatus
	}
	Bio        null.String
	IconFileID uuid.NullUUID
	LastOnline null.Time
	Password   null.String
}

// UsersQuery GetUsers用クエリ
type UsersQuery struct {
	IsBot                       null.Bool
	IsActive                    null.Bool
	IsCMemberOf                 uuid.NullUUID
	IsGMemberOf                 uuid.NullUUID
	IsSubscriberAtMarkLevelOf   uuid.NullUUID
	IsSubscriberAtNotifyLevelOf uuid.NullUUID
	EnableProfileLoading        bool
}

// NotBot Botでない
func (q UsersQuery) NotBot() UsersQuery {
	q.IsBot = null.BoolFrom(false)
	return q
}

// Active アカウントが有効である
func (q UsersQuery) Active() UsersQuery {
	q.IsActive = null.BoolFrom(true)
	return q
}

// CMemberOf channelIDプライベートチャンネルのメンバーである
func (q UsersQuery) CMemberOf(channelID uuid.UUID) UsersQuery {
	q.IsCMemberOf = uuid.NullUUID{
		UUID:  channelID,
		Valid: true,
	}
	return q
}

// GMemberOf groupIDグループのメンバーである
func (q UsersQuery) GMemberOf(groupID uuid.UUID) UsersQuery {
	q.IsGMemberOf = uuid.NullUUID{
		UUID:  groupID,
		Valid: true,
	}
	return q
}

// SubscriberAtMarkLevelOf channelIDチャンネルの未読管理レベル購読ユーザーである
func (q UsersQuery) SubscriberAtMarkLevelOf(channelID uuid.UUID) UsersQuery {
	q.IsSubscriberAtMarkLevelOf = uuid.NullUUID{
		UUID:  channelID,
		Valid: true,
	}
	return q
}

// SubscriberAtNotifyLevelOf channelIDチャンネルの通知レベル購読ユーザーである
func (q UsersQuery) SubscriberAtNotifyLevelOf(channelID uuid.UUID) UsersQuery {
	q.IsSubscriberAtNotifyLevelOf = uuid.NullUUID{
		UUID:  channelID,
		Valid: true,
	}
	return q
}

// LoadProfile ユーザーの追加プロファイル情報を読み込むかどうか
func (q UsersQuery) LoadProfile() UsersQuery {
	q.EnableProfileLoading = true
	return q
}

// UserRepository ユーザーリポジトリ
type UserRepository interface {
	// CreateUser ユーザーを作成します
	//
	// 成功した場合、ユーザーとnilを返します。
	// DBによるエラーを返すことがあります。
	CreateUser(name, password, role string) (model.UserInfo, error)
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
}
