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
	// 無効なパスワード文字列を指定した場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserPassword(id uuid.UUID, password string) error
	// ChangeUserIcon 指定したユーザーのアイコンを変更します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserIcon(id, fileID uuid.UUID) error
	// ChangeUserAccountStatus 指定したユーザーのアカウント状態を変更します
	//
	// 成功した場合、nilを返します。
	// 存在しないユーザーの場合、ErrNotFoundを返します。
	// 無効なstatusを指定した場合、ArgumentErrorを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error
	// UpdateUserLastOnline 指定したユーザーの最終オンライン日時を更新します
	//
	// 成功した場合、nilを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateUserLastOnline(id uuid.UUID, time time.Time) (err error)
	// IsUserOnline 指定したユーザーがオンラインかどうかを返します
	//
	// オンラインの場合、trueを返します。
	IsUserOnline(id uuid.UUID) bool
	// GetUserLastOnline 指定したユーザーの最終オンライン日時を取得します
	//
	// 成功した場合、日時とnilを返します。
	// 存在しないユーザーの場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetUserLastOnline(id uuid.UUID) (time.Time, error)
	// GetHeartbeatStatus 指定したチャンネルのHeartbeatStatusを取得します
	//
	// 成功した場合、HeartbeatStatusとtrueを返します。
	// 失敗した場合、falseを返します。
	GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool)
	// UpdateHeartbeatStatus 指定したユーザーのハートビートを更新します
	UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string)
}
