package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// ChannelCreated CHANNEL_CREATEDイベントペイロード
type ChannelCreated struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeChannelCreated(eventTime time.Time, ch *model.Channel, chPath string, user model.UserInfo) *ChannelCreated {
	return &ChannelCreated{
		Base:    MakeBase(eventTime),
		Channel: MakeChannel(ch, chPath, user),
	}
}
