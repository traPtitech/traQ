package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// UserActivated USER_ACTIVATEDイベントペイロード
type UserActivated struct {
	Base
	User User `json:"user"`
}

func MakeUserActivated(et time.Time, user model.UserInfo) *UserActivated {
	return &UserActivated{
		Base: MakeBase(et),
		User: MakeUser(user),
	}
}
