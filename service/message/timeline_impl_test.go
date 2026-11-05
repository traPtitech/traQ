package message

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/model"
)

func TestTimeline_Records(t *testing.T) {
	t.Parallel()

	t.Run("with thread", func(t *testing.T) {
		t.Parallel()

		threadID := uuid.NewV3(uuid.Nil, "thread")
		tl := &timeline{
			records: []*model.Message{{
				Thread: &model.Thread{ChannelID: threadID},
			}},
			preloaded: true,
		}

		data, err := json.Marshal(tl.Records()[0])
		if assert.NoError(t, err) {
			var response map[string]interface{}
			if assert.NoError(t, json.Unmarshal(data, &response)) {
				assert.EqualValues(t, threadID.String(), response["threadId"])
			}
		}
	})

	t.Run("without thread", func(t *testing.T) {
		t.Parallel()

		tl := &timeline{
			records:   []*model.Message{{}},
			preloaded: true,
		}

		data, err := json.Marshal(tl.Records()[0])
		if assert.NoError(t, err) {
			var response map[string]interface{}
			if assert.NoError(t, json.Unmarshal(data, &response)) {
				assert.Contains(t, response, "threadId")
				assert.Nil(t, response["threadId"])
			}
		}
	})

	t.Run("without preload", func(t *testing.T) {
		t.Parallel()

		tl := &timeline{
			records: []*model.Message{{
				Thread: &model.Thread{ChannelID: uuid.NewV3(uuid.Nil, "thread")},
			}},
			preloaded: false,
		}

		data, err := json.Marshal(tl.Records()[0])
		if assert.NoError(t, err) {
			var response map[string]interface{}
			if assert.NoError(t, json.Unmarshal(data, &response)) {
				assert.NotContains(t, response, "threadId")
			}
		}
	})
}
