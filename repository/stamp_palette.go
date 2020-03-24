package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/guregu/null.v3"
)

// UpdateStampPaletteArgs スタンプパレット情報更新引数
type UpdateStampPaletteArgs struct {
	Name        null.String
	Description null.String
	Stamps      model.UUIDs
}

// StampPaletteRepository スタンプパレットリポジトリ
type StampPaletteRepository interface {
	// CreateStampPalette スタンプパレットを作成します
	//
	// 成功した場合、スタンプパレットとnilを返します。
	// userIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	CreateStampPalette(name, description string, stamps model.UUIDs, userID uuid.UUID) (sp *model.StampPalette, err error)
	// UpdateStampPalette 指定したスタンプパレットの情報を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しないスタンプパレットの場合、ErrNotFoundを返します。
	// idにuuid.Nilを指定した場合、ErrNilIDを返します。
	// 更新内容に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	UpdateStampPalette(id uuid.UUID, args UpdateStampPaletteArgs) error
	// GetStampPalette 指定したIDのスタンプパレットを取得します
	//
	// 成功した場合、スタンプパレットとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetStampPalette(id uuid.UUID) (sp *model.StampPalette, err error)
	// DeleteStampPalette 指定したIDのスタンプパレットを削除します
	//
	// 成功した場合、nilを返します。
	// 既に存在しない場合、ErrNotFoundを返します。
	// 引数にuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	DeleteStampPalette(id uuid.UUID) (err error)
	// GetStampPalettes そのユーザーが作成したスタンプパレットを取得します
	//
	// 成功した場合、スタンプパレットの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetStampPalettes(userID uuid.UUID) (sps []*model.StampPalette, err error)
}
