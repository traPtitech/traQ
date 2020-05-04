package utils

import (
	"context"
	"errors"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
)

type ctxKey int

const (
	repoCtxKey ctxKey = iota
)

func NewRequestValidateContext(c echo.Context) context.Context {
	return context.WithValue(context.Background(), repoCtxKey, c.Get(consts.KeyRepo))
}

// IsPublicChannelID 公開チャンネルのUUIDである
var IsPublicChannelID = vd.WithContext(func(ctx context.Context, value interface{}) error {
	const errMessage = "invalid channel id"

	repo, ok := ctx.Value(repoCtxKey).(repository.Repository)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have repository"))
	}

	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		if !repo.GetChannelTree().IsChannelPresent(v) {
			return errors.New(errMessage)
		}
	case uuid.NullUUID:
		if v.Valid && !repo.GetChannelTree().IsChannelPresent(v.UUID) {
			return errors.New(errMessage)
		}
	case string:
		if !repo.GetChannelTree().IsChannelPresent(uuid.FromStringOrNil(v)) {
			return errors.New(errMessage)
		}
	case []byte:
		if !repo.GetChannelTree().IsChannelPresent(uuid.FromBytesOrNil(v)) {
			return errors.New(errMessage)
		}
	default:
		return errors.New(errMessage)
	}
	return nil
})
