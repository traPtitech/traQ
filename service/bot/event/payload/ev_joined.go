package payload

import (
	"github.com/traPtitech/traQ/model"
	"time"
)

// Joined JOINEDイベントペイロード
type Joined struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeJoined(et time.Time, ch *model.Channel, chPath string, user model.UserInfo) *Joined {
	return &Joined{
		Base:    MakeBase(et),
		Channel: MakeChannel(ch, chPath, user),
	}
}
