package event

import (
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event/mock_event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"testing"
)

func TestUnicast(t *testing.T) {
	t.Parallel()

	t.Run("no target", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		d := mock_event.NewMockDispatcher(ctrl)

		assert.NoError(t, Unicast(d, Ping, payload.MakePing(), nil))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		d := mock_event.NewMockDispatcher(ctrl)

		b1 := &model.Bot{ID: uuid.NewV3(uuid.Nil, "b1")}
		p := payload.MakePing()
		body, release, _ := makePayloadJSON(p)
		defer release()

		d.EXPECT().
			Send(b1, Ping, body).
			Return(true).
			Times(1)

		assert.NoError(t, Unicast(d, Ping, p, b1))
	})

}

func TestMulticast(t *testing.T) {
	t.Parallel()

	t.Run("no target", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		d := mock_event.NewMockDispatcher(ctrl)

		assert.NoError(t, Multicast(d, Ping, payload.MakePing(), []*model.Bot{}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		d := mock_event.NewMockDispatcher(ctrl)

		b1 := &model.Bot{ID: uuid.NewV3(uuid.Nil, "b1")}
		b2 := &model.Bot{ID: uuid.NewV3(uuid.Nil, "b2")}
		b3 := &model.Bot{ID: uuid.NewV3(uuid.Nil, "b3")}
		bots := []*model.Bot{b1, b2, b1, b3}
		p := payload.MakePing()
		body, release, _ := makePayloadJSON(p)
		defer release()

		d.EXPECT().
			Send(b1, Ping, body).
			Return(true).
			Times(1)
		d.EXPECT().
			Send(b2, Ping, body).
			Return(true).
			Times(1)
		d.EXPECT().
			Send(b3, Ping, body).
			Return(true).
			Times(1)

		assert.NoError(t, Multicast(d, Ping, p, bots))
	})
}
