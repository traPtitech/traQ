package payload

import (
	"github.com/traPtitech/traQ/model"
	"time"
)

// UserCreated USER_CREATEDイベントペイロード
type UserCreated struct {
	Base
	User User `json:"user"`
}

func MakeUserCreated(et time.Time, user model.UserInfo) *UserCreated {
	return &UserCreated{
		Base: MakeBase(et),
		User: MakeUser(user),
	}
}
