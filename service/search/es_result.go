package search

import (
	"github.com/gofrs/uuid"
	"github.com/olivere/elastic/v7"

	"github.com/traPtitech/traQ/service/message"
)

// esResult search.Result 実装
type esResult struct {
	totalHits int64
	messages  []message.Message
}

func (e *esEngine) bindESResult(sr *elastic.SearchResult) (Result, error) {
	r := &esResult{
		totalHits: sr.TotalHits(),
		messages:  make([]message.Message, 0, len(sr.Hits.Hits)),
	}

	for _, hit := range sr.Hits.Hits {
		// NOTE: N+1 の可能性
		m, err := e.mm.Get(uuid.Must(uuid.FromString(hit.Id)))
		if err != nil {
			return nil, err
		}
		r.messages = append(r.messages, m)
	}

	return r, nil
}

func (e *esResult) TotalHits() int64 {
	return e.totalHits
}

func (e *esResult) Hits() []message.Message {
	return e.messages
}
