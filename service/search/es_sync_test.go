package search

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/utils/message"
)

func TestEsEngine_convertMessageCreated_IsPublic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		channelType model.ChannelType
		want        bool
	}{
		{name: "public channel", channelType: model.ChannelTypePublic, want: true},
		{name: "thread channel", channelType: model.ChannelTypeThread, want: true},
		{name: "dm channel", channelType: model.ChannelTypeDM, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &model.Message{
				ID:        uuid.Must(uuid.NewV4()),
				UserID:    uuid.Must(uuid.NewV4()),
				ChannelID: uuid.Must(uuid.NewV4()),
				Text:      "test",
			}

			cm := mock_channel.NewMockManager(gomock.NewController(t))
			cm.EXPECT().
				GetChannel(gomock.Any(), m.ChannelID).
				Return(&model.Channel{ID: m.ChannelID, Type: tt.channelType}, nil)

			e := &esEngine{cm: cm}
			doc, err := e.convertMessageCreated(m, message.Parse(m.Text), userCache{m.UserID: false})
			require.NoError(t, err)
			assert.Equal(t, tt.want, doc.IsPublic)
		})
	}

	t.Run("channel not found", func(t *testing.T) {
		t.Parallel()

		m := &model.Message{
			ID:        uuid.Must(uuid.NewV4()),
			UserID:    uuid.Must(uuid.NewV4()),
			ChannelID: uuid.Must(uuid.NewV4()),
			Text:      "test",
		}

		cm := mock_channel.NewMockManager(gomock.NewController(t))
		cm.EXPECT().
			GetChannel(gomock.Any(), m.ChannelID).
			Return(nil, channel.ErrChannelNotFound)

		e := &esEngine{cm: cm}
		doc, err := e.convertMessageCreated(m, message.Parse(m.Text), userCache{m.UserID: false})
		require.ErrorIs(t, err, channel.ErrChannelNotFound)
		assert.Nil(t, doc)
	})
}
