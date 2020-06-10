package payload

import "github.com/traPtitech/traQ/model"

// Joined JOINEDイベントペイロード
type Joined struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeJoined(ch *model.Channel, chPath string, user model.UserInfo) *Joined {
	return &Joined{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, user),
	}
}
