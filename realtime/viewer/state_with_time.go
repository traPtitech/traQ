package viewer

import "time"

// StateWithTime 閲覧状態
type StateWithTime struct {
	State State
	Time  time.Time
}
