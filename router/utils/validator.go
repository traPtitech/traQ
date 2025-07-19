// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
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
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/utils/optional"
)

type ctxKey int

const (
	repoctxKey ctxKey = iota
	cmctxKey
)

func NewRequestValidateContext(c echo.Context) context.Context {
	return context.WithValue(context.WithValue(context.Background(), repoctxKey, c.Get(consts.KeyRepo)), cmctxKey, c.Get(consts.KeyChannelManager))
}

// IsPublicChannelID 公開チャンネルのUUIDである
var IsPublicChannelID = vd.WithContext(func(ctx context.Context, value interface{}) error {
	const errMessage = "invalid channel id"

	cm, ok := ctx.Value(cmctxKey).(channel.Manager)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have ChannelManager"))
	}

	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		if !cm.IsPublicChannel(v) {
			return errors.New(errMessage)
		}
	case optional.Of[uuid.UUID]:
		if v.Valid && !cm.IsPublicChannel(v.V) {
			return errors.New(errMessage)
		}
	case string:
		if !cm.IsPublicChannel(uuid.FromStringOrNil(v)) {
			return errors.New(errMessage)
		}
	case []byte:
		if !cm.IsPublicChannel(uuid.FromBytesOrNil(v)) {
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

	repo, ok := ctx.Value(repoctxKey).(repository.Repository)
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
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		u, err = repo.GetUser(v.V, false)
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

	repo, ok := ctx.Value(repoctxKey).(repository.Repository)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have repository"))
	}

	var err error
	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		ok, err = repo.UserExists(v)
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		ok, err = repo.UserExists(v.V)
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

// IsNotWebhookUserID WebhookのユーザーIDではない
var IsNotWebhookUserID = vd.WithContext(func(ctx context.Context, value interface{}) error {
	const errMessage = "invalid user id"

	repo, ok := ctx.Value(repoctxKey).(repository.Repository)
	if !ok {
		return vd.NewInternalError(errors.New("this context didn't have repository"))
	}

	var (
		user model.UserInfo
		err  error
	)
	switch v := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		user, err = repo.GetUser(v, false)
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		user, err = repo.GetUser(v.V, false)
	case string:
		user, err = repo.GetUser(uuid.FromStringOrNil(v), false)
	case []byte:
		user, err = repo.GetUser(uuid.FromBytesOrNil(v), false)
	default:
		return errors.New(errMessage)
	}
	if err != nil {
		if err == repository.ErrNotFound {
			return nil
		}
		return vd.NewInternalError(err)
	}

	if user.GetUserType() == model.UserTypeWebhook {
		return errors.New(errMessage)
	}

	return nil
})

// IsValidBotEvents 有効なBOTイベントのセットである
var IsValidBotEvents = vd.By(func(value interface{}) error {
	s, ok := value.(model.BotEventTypes)
	if !ok || s == nil {
		return nil
	}
	for v := range s {
		if !event.Types.Contains(v) {
			return errors.New("must be valid bot event type")
		}
	}
	return nil
})
