package model

import (
	"errors"
	"github.com/satori/go.uuid"
)

var (
	// ErrForbidden 禁止されている操作です
	ErrForbidden = errors.New("forbidden")
)

// Mute ミュートチャンネルのレコード
type Mute struct {
	UserID    string `gorm:"type:char(36);primary_key"`
	ChannelID string `gorm:"type:char(36);primary_key"`
}

// TableName Mute構造体のテーブル名
func (m *Mute) TableName() string {
	return "mutes"
}

// MuteChannel 指定したチャンネルをミュートします
func MuteChannel(userID, channelID uuid.UUID) error {
	// ユーザーからチャンネルが見えるかどうか
	ch, err := GetChannelWithUserID(userID, channelID)
	if err != nil {
		if err == ErrNotFoundOrForbidden {
			return ErrNotFound
		}
		return err
	}

	// 強制通知チャンネルはミュート不可
	if ch.IsForced {
		return ErrForbidden
	}

	if muted, err := IsChannelMuted(userID, channelID); err != nil {
		return err
	} else if !muted {
		if err := db.Create(&Mute{UserID: userID.String(), ChannelID: channelID.String()}).Error; err != nil {
			if isMySQLDuplicatedRecordErr(err) {
				return nil
			}
			return err
		}
	}

	return nil
}

// UnmuteChannel 指定したチャンネルをアンミュートします
func UnmuteChannel(userID, channelID uuid.UUID) error {
	// ユーザーからチャンネルが見えるかどうか
	_, err := GetChannelWithUserID(userID, channelID)
	if err != nil {
		if err == ErrNotFoundOrForbidden {
			return ErrNotFound
		}
		return err
	}

	if err := db.Delete(&Mute{UserID: userID.String(), ChannelID: channelID.String()}).Error; err != nil {
		return err
	}

	return nil
}

// GetMutedChannelIDs ミュートしているチャンネルのIDの配列を取得します
func GetMutedChannelIDs(userID uuid.UUID) (ids []string, err error) {
	ids = make([]string, 0)
	if err = db.Model(Mute{}).Where(&Mute{UserID: userID.String()}).Pluck("channel_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// GetMuteUserIDs ミュートしているユーザーのIDの配列を取得します
func GetMuteUserIDs(channelID uuid.UUID) (ids []string, err error) {
	ids = make([]string, 0)
	if err = db.Model(Mute{}).Where(&Mute{ChannelID: channelID.String()}).Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// IsChannelMuted 指定したユーザーが指定したチャンネルをミュートしているかどうかを返します
func IsChannelMuted(userID, channelID uuid.UUID) (muted bool, err error) {
	c := 0
	if err := db.Model(Mute{}).Where(&Mute{UserID: userID.String(), ChannelID: channelID.String()}).Count(&c).Error; err != nil {
		return false, err
	}
	return c == 1, nil
}
