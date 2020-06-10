package payload

import (
	"github.com/traPtitech/traQ/model"
)

// Left LEFTイベントペイロード
type Left struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeLeft(ch *model.Channel, chPath string, user model.UserInfo) *Left {
	return &Left{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, user),
	}
}
