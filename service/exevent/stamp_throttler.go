package exevent

import (
	"sync"
	"time"

	"github.com/boz/go-throttle"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/service/message"
)

func NewStampThrottler(bus *hub.Hub, mm message.Manager) *StampThrottler {
	st := &StampThrottler{
		bus: bus,
		mm:  mm,
		m:   map[uuid.UUID]*stampThrottlerEntry{},
	}
	return st
}

type StampThrottler struct {
	bus   *hub.Hub
	mm    message.Manager
	m     map[uuid.UUID]*stampThrottlerEntry
	mLock sync.Mutex
}

func (st *StampThrottler) Start() {
	go st.loop()
}

func (st *StampThrottler) loop() {
	clean := time.NewTicker(5 * time.Second)
	defer clean.Stop()
	sub := st.bus.Subscribe(100, event.MessageStamped, event.MessageUnstamped)
	defer st.bus.Unsubscribe(sub)

	for {
		select {
		case msg := <-sub.Receiver:
			mid := msg.Fields["message_id"].(uuid.UUID)
			st.mLock.Lock()
			ent, ok := st.m[mid]
			if !ok {
				ent = &stampThrottlerEntry{
					t: throttle.ThrottleFunc(time.Second, true, func() {
						m, err := st.mm.Get(mid)
						if err != nil {
							return // 無視
						}

						st.bus.Publish(hub.Message{
							Name: event.MessageStampsUpdated,
							Fields: hub.Fields{
								"message_id": mid,
								"message":    m,
							},
						})
					}),
				}
				st.m[mid] = ent
			}
			ent.trigger()
			st.mLock.Unlock()

		case <-clean.C:
			st.mLock.Lock()
			for mid, ent := range st.m {
				if ent.lastCall.Add(10 * time.Second).Before(time.Now()) {
					ent.dispose()
					delete(st.m, mid)
				}
			}
			st.mLock.Unlock()
		}
	}
}

type stampThrottlerEntry struct {
	lastCall time.Time
	t        throttle.ThrottleDriver
}

func (ste *stampThrottlerEntry) dispose() {
	ste.t.Stop()
}

func (ste *stampThrottlerEntry) trigger() {
	ste.lastCall = time.Now()
	ste.t.Trigger()
}
