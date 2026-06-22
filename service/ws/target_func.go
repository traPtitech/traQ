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
func TargetUserSets(sets ...set.UUID) TargetFunc {
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

// TargetChannelViewersの複数形
func TargetChannelsViewers(channelIDs []uuid.UUID) TargetFunc {
	return func(s Session) bool {
		c, _ := s.ViewState()
		for _, id := range channelIDs {
			if c == id {
				return true
			}
		}
		return false
	}
}

// TargetTimelineStreamingEnabled タイムラインストリーミングが有効なコネクションを対象に送信します
func TargetTimelineStreamingEnabled() TargetFunc {
	return func(s Session) bool {
		return s.TimelineStreaming()
	}
}

// TargetNone いずれのセッションにも送信しません
func TargetNone() TargetFunc {
	return func(_ Session) bool {
		return false
	}
}

// Or いずれかのTargetFuncの条件に該当する対象に送信します
func Or(funcs ...TargetFunc) TargetFunc {
	return func(s Session) bool {
		for _, f := range funcs {
			if f(s) {
				return true
			}
		}
		return false
	}
}

// And すべてのTargetFuncの条件に該当する対象に送信します
func And(funcs ...TargetFunc) TargetFunc {
	return func(s Session) bool {
		for _, f := range funcs {
			if !f(s) {
				return false
			}
		}
		return true
	}
}

// Not TargetFuncの条件に該当しない対象に送信します
func Not(f TargetFunc) TargetFunc {
	return func(s Session) bool {
		return !f(s)
	}
}
