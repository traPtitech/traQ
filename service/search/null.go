package search

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

var nullE = &nullEngine{}

type nullEngine struct{}

// NewNullEngine 常に利用不可な検索エンジンを返します
func NewNullEngine() Engine {
	return nullE
}

func (n *nullEngine) Do(*Query) (Result, error) {
	return nil, ErrServiceUnavailable
}

func (n *nullEngine) Available() bool {
	return false
}

func (n *nullEngine) Close() error {
	return nil
}

func (n *nullEngine) ProcessImagesForMessages(context.Context, []*model.Message) error {
	return ErrServiceUnavailable
}

func (n *nullEngine) GetUnprocessedImageMessageIDs(context.Context) ([]uuid.UUID, error) {
	return nil, ErrServiceUnavailable
}

func (n *nullEngine) ClearImageIndex(context.Context) error {
	return ErrServiceUnavailable
}
