package viewer

import (
	"github.com/gofrs/uuid"
	"sort"
	"time"
)

// UserState ユーザー閲覧状態
type UserState struct {
	UserID uuid.UUID `json:"userId"`
	State  State     `json:"state"`
	time   time.Time
}

// UserStates []UserState
type UserStates []UserState

// Len implements sort.Interface
func (u UserStates) Len() int {
	return len(u)
}

// Less implements sort.Interface
func (u UserStates) Less(i, j int) bool {
	return u[i].time.Before(u[j].time)
}

// Swap implements sort.Interface
func (u UserStates) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

// ConvertToArray 閲覧状態mapをsliceに変換します
func ConvertToArray(cv map[uuid.UUID]StateWithTime) UserStates {
	result := make(UserStates, 0, len(cv))
	for uid, swt := range cv {
		result = append(result, UserState{
			UserID: uid,
			State:  swt.State,
			time:   swt.Time,
		})
	}
	sort.Sort(result)
	return result
}
