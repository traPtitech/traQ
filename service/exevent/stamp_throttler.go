package exevent

import (
	"context"
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
	throttles *throttle.Map[uuid.UUID]
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
		var (
			messageID uuid.UUID
			ok        bool
		)
		switch msg.Name {
		case event.MessageStamped:
			var p event.MessageStampedPayload
			p, ok = event.PayloadOf[event.MessageStampedPayload](msg.Fields)
			messageID = p.MessageID
		case event.MessageUnstamped:
			var p event.MessageUnstampedPayload
			p, ok = event.PayloadOf[event.MessageUnstampedPayload](msg.Fields)
			messageID = p.MessageID
		default:
			panic("unexpected topic: " + msg.Name)
		}
		if !ok {
			continue // ignore invalid message
		}
		st.throttles.Trigger(messageID)
	}
}

func (st *StampThrottler) publishMessageStampsUpdated(messageID uuid.UUID) {
	msg, err := st.mm.Get(context.Background(), messageID)
	if err != nil {
		return // ignore error
	}

	event.Publish(st.bus, MessageStampsUpdatedPayload{
		MessageID: messageID,
		Message:   msg,
	})
}
