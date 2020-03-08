package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
	"time"
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
	Bio null.String
}

// UsersQuery GetUsers用クエリ
type UsersQuery struct {
	IsBot          null.Bool
	IsActive       null.Bool
	IsCMemberOf    uuid.NullUUID
	IsGMemberOf    uuid.NullUUID
	IsSubscriberOf uuid.NullUUID
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

// SubscriberOf channelIDチャンネルの購読者である
func (q UsersQuery) SubscriberOf(channelID uuid.UUID) UsersQuery {
	q.IsSubscriberOf = uuid.NullUUID{
		UUID:  channelID,
		Valid: true,
	}
	return q
}

// UserRepository ユーザーリポジトリ
type UserRepository interface {
	// CreateUser ユーザーを作成します
	//
	// 成功した場合、ユーザーとnilを返します。
	// DBによるエラーを返すことがあります。
	CreateUser(name, password, role string) (*model.User, error)
	// GetUser 指定したIDのユーザーを取得します
	//
	// 成功した場合、ユーザーとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUser(id uuid.UUID) (*model.User, error)
	// GetUserByName 指定したNameのユーザーを取得する
	//
	// 成功した場合、ユーザーとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserByName(name string) (*model.User, error)
	// GetUsers 全ユーザーを取得します
	//
	// 成功した場合、ユーザーの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUsers() ([]*model.User, error)
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
	// ChangeUserPassword 指定したユーザーのパスワードを変更します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserPassword(id uuid.UUID, password string) error
	// ChangeUserIcon 指定したユーザーのアイコンを変更します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserIcon(id, fileID uuid.UUID) error
	// UpdateUserLastOnline 指定したユーザーの最終オンライン日時を更新します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error)
}
