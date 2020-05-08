package utils

import (
	"context"
	"errors"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/utils/optional"
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
	case optional.UUID:
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

// IsActiveHumanUserID アカウントが有効な一般ユーザーのUUIDである
var IsActiveHumanUserID = vd.WithContext(func(ctx context.Context, value interface{}) error {
	const errMessage = "invalid user id"

	repo, ok := ctx.Value(repoCtxKey).(repository.Repository)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have repository"))
	}

	var (
		u   model.UserInfo
		err error
	)
	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		u, err = repo.GetUser(v, false)
	case optional.UUID:
		if !v.Valid {
			return nil
		}
		u, err = repo.GetUser(v.UUID, false)
	case string:
		u, err = repo.GetUser(uuid.FromStringOrNil(v), false)
	case []byte:
		u, err = repo.GetUser(uuid.FromBytesOrNil(v), false)
	default:
		return errors.New(errMessage)
	}
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return errors.New(errMessage)
		default:
			return vd.NewInternalError(err)
		}
	}

	if !u.IsActive() || u.IsBot() {
		return errors.New(errMessage)
	}

	return nil
})

// IsUserID ユーザーのUUIDである
var IsUserID = vd.WithContext(func(ctx context.Context, value interface{}) error {
	const errMessage = "invalid user id"

	repo, ok := ctx.Value(repoCtxKey).(repository.Repository)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have repository"))
	}

	var err error
	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		ok, err = repo.UserExists(v)
	case optional.UUID:
		if !v.Valid {
			return nil
		}
		ok, err = repo.UserExists(v.UUID)
	case string:
		ok, err = repo.UserExists(uuid.FromStringOrNil(v))
	case []byte:
		ok, err = repo.UserExists(uuid.FromBytesOrNil(v))
	default:
		return errors.New(errMessage)
	}
	if err != nil {
		return vd.NewInternalError(err)
	}
	if !ok {
		return errors.New(errMessage)
	}
	return nil
})
