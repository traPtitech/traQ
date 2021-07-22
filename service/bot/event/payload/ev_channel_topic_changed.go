package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// ChannelTopicChanged CHANNEL_TOPIC_CHANGEDイベントペイロード
type ChannelTopicChanged struct {
	Base
	Channel Channel `json:"channel"`
	Topic   string  `json:"topic"`
	Updater User    `json:"updater"`
}

func MakeChannelTopicChanged(et time.Time, ch *model.Channel, chPath string, chCreator model.UserInfo, topic string, user model.UserInfo) *ChannelTopicChanged {
	return &ChannelTopicChanged{
		Base:    MakeBase(et),
		Channel: MakeChannel(ch, chPath, chCreator),
		Topic:   topic,
		Updater: MakeUser(user),
	}
}
