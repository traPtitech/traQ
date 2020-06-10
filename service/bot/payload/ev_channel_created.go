package payload

import "github.com/traPtitech/traQ/model"

// ChannelCreated CHANNEL_CREATEDイベントペイロード
type ChannelCreated struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeChannelCreated(ch *model.Channel, chPath string, user model.UserInfo) *ChannelCreated {
	return &ChannelCreated{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, user),
	}
}
