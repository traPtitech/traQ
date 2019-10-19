package ws

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/set"
)

// TargetFunc メッセージ送信対象関数
type TargetFunc func(s Session) bool

// TargetAll 全セッションを対象に送信します
func TargetAll() TargetFunc {
	return func(_ Session) bool {
		return true
	}
}

// TargetUsers 指定したユーザーを対象に送信します
func TargetUsers(userID ...uuid.UUID) TargetFunc {
	return func(s Session) bool {
		for _, u := range userID {
			if u == s.UserID() {
				return true
			}
		}
		return false
	}
}

// TargetUserSets 指定したユーザーを対象に送信します
func TargetUserSets(sets ...set.UUIDSet) TargetFunc {
	return func(s Session) bool {
		for _, set := range sets {
			if set.Contains(s.UserID()) {
				return true
			}
		}
		return false
	}
}

// TargetChannelViewers 指定したチャンネルの閲覧者を対象に送信します
func TargetChannelViewers(channelID uuid.UUID) TargetFunc {
	return func(s Session) bool {
		c, _ := s.ViewState()
		return c == channelID
	}
}
