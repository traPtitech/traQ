package viewer

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/event"
)

// UserViewStateChangedPayload UserViewStateChanged のペイロード
//
// ペイロード型は event package に置くのが原則だが、StateWithChannel を使うため
// import が循環しないよう例外的にここに置いている。
// TODO: PubSub に model 以外の構造体を使わないようにして、event package に分離できるようにする
type UserViewStateChangedPayload struct {
	UserID     uuid.UUID                   `hub:"user_id"`
	ViewStates map[string]StateWithChannel `hub:"view_states"`
}

func (UserViewStateChangedPayload) TopicName() string { return event.UserViewStateChanged }

// ChannelViewersChangedPayload ChannelViewersChanged のペイロード
//
// ペイロード型は event package に置くのが原則だが、StateWithTime を使うため
// import が循環しないよう例外的にここに置いている。
// TODO: PubSub に model 以外の構造体を使わないようにして、event package に分離できるようにする
type ChannelViewersChangedPayload struct {
	ChannelID uuid.UUID                   `hub:"channel_id"`
	Viewers   map[uuid.UUID]StateWithTime `hub:"viewers"`
}

func (ChannelViewersChangedPayload) TopicName() string { return event.ChannelViewersChanged }

var (
	_ event.Payload = UserViewStateChangedPayload{}
	_ event.Payload = ChannelViewersChangedPayload{}
)
