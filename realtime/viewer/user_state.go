package viewer

import "github.com/gofrs/uuid"

// UserState ユーザー閲覧状態
type UserState struct {
	UserID uuid.UUID `json:"userId"`
	State  State     `json:"state"`
}

// ConvertToArray 閲覧状態mapをsliceに変換します
func ConvertToArray(cv map[uuid.UUID]State) []UserState {
	result := make([]UserState, 0, len(cv))
	for uid, state := range cv {
		result = append(result, UserState{
			UserID: uid,
			State:  state,
		})
	}
	return result
}
