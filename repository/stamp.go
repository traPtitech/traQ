package repository

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/etag"
	"github.com/traPtitech/traQ/utils/optional"
)

// CreateStampArgs スタンプ作成引数
type CreateStampArgs struct {
	Name      string
	FileID    uuid.UUID
	CreatorID uuid.UUID
	IsUnicode bool
}

// UpdateStampArgs スタンプ情報更新引数
type UpdateStampArgs struct {
	Name      optional.Of[string]
	FileID    optional.Of[uuid.UUID]
	CreatorID optional.Of[uuid.UUID]
}

// UserStampHistory スタンプ履歴構造体
type UserStampHistory struct {
	StampID  uuid.UUID `json:"stampId"`
	Datetime time.Time `json:"datetime"`
}

// UserStampRecommendation スタンプレコメンデーション構造体
type UserStampRecommendation struct {
	StampID uuid.UUID `json:"stampId"`
	Score   float64   `json:"score"`
}

// StampStats スタンプ統計情報
type StampStats struct {
	Count      int64 `json:"count"`
	TotalCount int64 `json:"totalCount"`
}

// StampType スタンプの種類
type StampType string

const (
	// StampTypeUnicode Unicodeスタンプ
	StampTypeUnicode StampType = "unicode"
	// StampTypeOriginal オリジナルスタンプ
	StampTypeOriginal StampType = "original"
	// StampTypeAll 全てのスタンプ
	StampTypeAll StampType = "all"
)

// StampRepository スタンプリポジトリ
type StampRepository interface {
	// CreateStamp スタンプを作成します
	//
	// 成功した場合、スタンプとnilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// 既にNameが使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	CreateStamp(ctx context.Context, args CreateStampArgs) (s *model.Stamp, err error)
	// UpdateStamp 指定したスタンプの情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないスタンプの場合、ErrNotFoundを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// 変更後のNameが既に使われている場合、ErrAlreadyExistsを返します。
	// DBによるエラーを返すことがあります。
	UpdateStamp(ctx context.Context, id uuid.UUID, args UpdateStampArgs) error
	// GetStamp 指定したIDのスタンプを取得します
	//
	// 成功した場合、スタンプとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetStamp(ctx context.Context, id uuid.UUID) (s *model.Stamp, err error)
	// GetStampByName 指定したnameのスタンプを取得します
	//
	// 成功した場合、スタンプとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetStampByName(ctx context.Context, name string) (s *model.Stamp, err error)
	// DeleteStamp 指定したIDのスタンプを削除します
	//
	// 成功した場合、nilを返します。
	// 既に存在しない場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteStamp(ctx context.Context, id uuid.UUID) (err error)
	// GetAllStampsWithThumbnail 全てのスタンプとサムネイルの有無を取得します
	//
	// 成功した場合、スタンプのIDでソートされた配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllStampsWithThumbnail(ctx context.Context, stampType StampType) (stampsWithETag *etag.Entity[[]*model.StampWithThumbnail], err error)
	// StampExists 指定したIDのスタンプが存在するかどうかを返します
	//
	// 存在する場合、trueとnilを返します。
	// DBによるエラーを返すことがあります。
	StampExists(ctx context.Context, id uuid.UUID) (bool, error)
	// GetUserStampHistory 指定したユーザーのスタンプ履歴を最大limit件取得します
	//
	// limitに負の値を指定した場合、全て取得します。
	// 成功した場合、降順のスタンプ履歴の配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserStampHistory(ctx context.Context, userID uuid.UUID, limit int) (h []*UserStampHistory, err error)
	// GetUserStampRecommendations 指定したユーザーのスタンプレコメンドを最大limit件取得します
	//
	// limitに負の値を指定した場合、全て取得します。
	// 成功した場合、レコメンドスコアの高い順で並んだスタンプIDの配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetUserStampRecommendations(ctx context.Context, userID uuid.UUID, limit int) (h []*UserStampRecommendation, err error)
	// ExistStamps stampIDの配列から指定したスタンプが全て存在するか判定します
	//
	// 成功した場合、nilを返します。
	// 存在しないスタンプがあった場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	ExistStamps(ctx context.Context, stampIDs []uuid.UUID) (err error)
	// GetStampStats 成功した場合、(統計情報, nil)を返します。
	//
	// スタンプがない場合、(nil, ErrNotFound)を返します。
	// stampIDにNILを渡した場合、(nil, ErrNilID)を返します。
	// DBによるエラーを返すことがあります。
	GetStampStats(ctx context.Context, stampID uuid.UUID) (*StampStats, error)
}
