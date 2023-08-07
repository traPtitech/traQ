package search

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/samber/lo"

	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils"
)

// esResult search.Result 実装
type esResult struct {
	totalHits int64
	messages  []message.Message
}

type esSearchResponse struct {
	Shards struct {
		Failed     int64 `json:"failed"`
		Skipped    int64 `json:"skipped"`
		Successful int64 `json:"successful"`
		Total      int64 `json:"total"`
	} `json:"_shards"`
	Hits struct {
		Hits []struct {
			ID     string       `json:"_id"`
			Index  string       `json:"_index"`
			Score  interface{}  `json:"_score"`
			Source esMessageDoc `json:"_source"`
			Type   string       `json:"_type"`
			Sort   []int64      `json:"sort"`
		} `json:"hits"`
		MaxScore interface{} `json:"max_score"`
		Total    struct {
			Relation string `json:"relation"`
			Value    int64  `json:"value"`
		} `json:"total"`
	} `json:"hits"`
	TimedOut bool  `json:"timed_out"`
	Took     int64 `json:"took"`
}

func (e *esEngine) parseResultBody(resBody m) (Result, error) {
	totalHits := resBody["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)
	hits := resBody["hits"].(map[string]interface{})["hits"].([]any)

	r := &esResult{
		totalHits: int64(totalHits),
		messages:  make([]message.Message, 0, len(hits)),
	}

	messageIDs := utils.Map(hits, func(hit any) uuid.UUID {
		return uuid.Must(uuid.FromString(hit.(map[string]any)["_id"].(string)))
	})

	messages, err := e.mm.GetIn(messageIDs)
	if err != nil {
		return nil, err
	}

	messagesMap := lo.SliceToMap(messages, func(m message.Message) (uuid.UUID, message.Message) {
		return m.GetID(), m
	})
	// sort result
	for _, id := range messageIDs {
		msg, ok := messagesMap[id]
		if !ok {
			return nil, fmt.Errorf("message %v not found", id)
		}
		r.messages = append(r.messages, msg)
	}

	return r, nil
}

func (e *esResult) TotalHits() int64 {
	return e.totalHits
}

func (e *esResult) Hits() []message.Message {
	return e.messages
}
