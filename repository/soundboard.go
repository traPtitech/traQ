//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// CreateSoundboardItemArgs サウンドボードアイテム作成引数
type CreateSoundboardItemArgs struct {
	SoundID   uuid.UUID
	SoundName string
	StampID   *uuid.UUID
	CreatorID uuid.UUID
}

// SoundboardRepository サウンドボードリポジトリ
type SoundboardRepository interface {
	// CreateSoundboardItem サウンドボードアイテムを作成します
	//
	// 成功した場合、nilを返します。
	// 引数に問題がある場合、ArgumentErrorを返します。
	// DBによるエラーを返すことがあります。
	CreateSoundboardItem(args CreateSoundboardItemArgs) error
	// GetAllSoundboardItems すべてのサウンドボードアイテムを取得します
	//
	// 成功した場合、サウンドボードアイテムの配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetAllSoundboardItems() ([]*model.SoundboardItem, error)
	// GetSoundboardItem 指定したIDのユーザーが作成したサウンドボードアイテムを取得します
	//
	// 成功した場合、サウンドボードアイテムとnilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetSoundboardByCreatorID(creatorID uuid.UUID) ([]*model.SoundboardItem, error)
	// UpdateSoundboardCreatorID サウンドボードアイテムの作成者を更新します
	//
	// 成功した場合、nilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	UpdateSoundboardCreatorID(soundID uuid.UUID, creatorID uuid.UUID) error
	// DeleteSoundboardItem サウンドボードアイテムを削除します
	//
	// 成功した場合、nilを返します。
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	DeleteSoundboardItem(soundID uuid.UUID) error
}
