package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// Left LEFTイベントペイロード
type Left struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeLeft(et time.Time, ch *model.Channel, chPath string, user model.UserInfo) *Left {
	return &Left{
		Base:    MakeBase(et),
		Channel: MakeChannel(ch, chPath, user),
	}
}
