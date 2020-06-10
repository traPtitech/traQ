package payload

import "github.com/traPtitech/traQ/model"

// UserCreated USER_CREATEDイベントペイロード
type UserCreated struct {
	Base
	User User `json:"user"`
}

func MakeUserCreated(user model.UserInfo) *UserCreated {
	return &UserCreated{
		Base: MakeBase(),
		User: MakeUser(user),
	}
}
