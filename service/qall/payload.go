package qall

import (
	"github.com/traPtitech/traQ/event"
)

// QallRoomStateChangedPayload QallRoomStateChanged のペイロード
//
// ペイロード型は event package に置くのが原則だが、RoomWithParticipants を使うため
// import が循環しないよう例外的にここに置いている。
// TODO: PubSub に model 以外の構造体を使わないようにして、event package に分離できるようにする
type QallRoomStateChangedPayload struct { //nolint:revive // あとで fix したいから #3601 で暫定的に入れている
	RoomStates []RoomWithParticipants `hub:"roomStates"`
}

func (QallRoomStateChangedPayload) TopicName() string { return event.QallRoomStateChanged }

var _ event.Payload = QallRoomStateChangedPayload{}
