package exevent

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/throttle"
)

const eventPublishInterval = 1 * time.Second

type StampThrottler struct {
	bus       *hub.Hub
	mm        message.Manager
	throttles *throttle.ThrottleMap[uuid.UUID]
}

func NewStampThrottler(bus *hub.Hub, mm message.Manager) *StampThrottler {
	st := &StampThrottler{
		bus: bus,
		mm:  mm,
	}
	st.throttles = throttle.NewThrottleMap(eventPublishInterval, 5*time.Second, st.publishMessageStampsUpdated)

	return st
}

func (st *StampThrottler) Start() {
	go st.run()
}

func (st *StampThrottler) run() {
	sub := st.bus.Subscribe(100, event.MessageStamped, event.MessageUnstamped)
	defer st.bus.Unsubscribe(sub)

	for msg := range sub.Receiver {
		messageID, ok := msg.Fields["message_id"].(uuid.UUID)
		if !ok {
			continue // ignore invalid message
		}
		st.throttles.Trigger(messageID)
	}
}

func (st *StampThrottler) publishMessageStampsUpdated(messageID uuid.UUID) {
	msg, err := st.mm.Get(messageID)
	if err != nil {
		return // ignore error
	}

	st.bus.Publish(hub.Message{
		Name: event.MessageStampsUpdated,
		Fields: hub.Fields{
			"message_id": messageID,
			"message":    msg,
		},
	})
}
