package viewer

import (
	"sort"
	"time"

	"github.com/gofrs/uuid"
)

// UserState ユーザー閲覧状態
type UserState struct {
	UserID    uuid.UUID `json:"userId"`
	State     State     `json:"state"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserStates []UserState
type UserStates []UserState

// Len implements sort.Interface
func (u UserStates) Len() int {
	return len(u)
}

// Less implements sort.Interface
func (u UserStates) Less(i, j int) bool {
	return u[i].UpdatedAt.Before(u[j].UpdatedAt)
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
			UserID:    uid,
			State:     swt.State,
			UpdatedAt: swt.Time,
		})
	}
	sort.Sort(result)
	return result
}
