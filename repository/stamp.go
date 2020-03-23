package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
	"time"
)

// UpdateStampArgs スタンプ情報更新引数
type UpdateStampArgs struct {
	Name      null.String
	FileID    uuid.NullUUID
	CreatorID uuid.NullUUID
}

// UserStampHistory スタンプ履歴構造体
type UserStampHistory struct {
	StampID  uuid.UUID `json:"stampId"`
	Datetime time.Time `json:"datetime"`
}

// StampRepository スタンプリポジトリ
type StampRepository interface {
	// CreateStamp スタンプを作成します
	//
	// 成功した場合、スタンプとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateStamp(name string, fileID, creatorID uuid.UUID) (s *model.Stamp, err error)
	// UpdateStamp 指定したスタンプの情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないスタンプの場合、ErrNotFoundを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// 変更後のNameが既に使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	UpdateStamp(id uuid.UUID, args UpdateStampArgs) error
	// GetStamp 指定したIDのスタンプを取得します
	//
	// 成功した場合、スタンプとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetStamp(id uuid.UUID) (s *model.Stamp, err error)
	// DeleteStamp 指定したIDのスタンプを削除します
	//
	// 成功した場合、nilを返します。
	// 既に存在しない場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteStamp(id uuid.UUID) (err error)
	// GetAllStamps 全てのスタンプを取得します
	//
	// 成功した場合、スタンプの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllStamps(excludeUnicode bool) (stamps []*model.Stamp, err error)
	// StampExists 指定したIDのスタンプが存在するかどうかを返します
	//
	// 存在する場合、trueとnilを返します。
	// DBによるエラーを返すことがあります。
	StampExists(id uuid.UUID) (bool, error)
	// StampNameExists 指定した名前のスタンプが存在するかどうかを返します
	//
	// 存在する場合、trueとnilを返します。
	// DBによるエラーを返すことがあります。
	StampNameExists(name string) (bool, error)
	// GetUserStampHistory 指定したユーザーのスタンプ履歴を最大limit件取得します
	//
	// 0を指定した場合、全て取得します。
	// 成功した場合、降順のスタンプ履歴の配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserStampHistory(userID uuid.UUID, limit int) (h []*UserStampHistory, err error)
	// ExistStamps stampIDの配列から指定したスタンプが全て存在するか判定します
	//
	// 成功した場合、nilを返します。
	// 存在しないスタンプがあった場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	ExistStamps(stampIDs model.UUIDs) (err error)
}
