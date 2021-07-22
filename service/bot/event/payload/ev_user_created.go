package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
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
