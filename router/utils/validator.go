// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
package utils

import (
	"context"
	"errors"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

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

func NewRequestValidateContext(c *echo.Context) context.Context {
	return context.WithValue(context.WithValue(c.Request().Context(), repoctxKey, c.Get(consts.KeyRepo)), cmctxKey, c.Get(consts.KeyChannelManager))
}

// IsPublicChannelID 公開チャンネルのUUIDである
var IsPublicChannelID = channelIDRule(func(ctx context.Context, cm channel.Manager, id uuid.UUID) (bool, error) {
	return cm.IsPublicChannel(ctx, id), nil
})

// IsPublicOrThreadChannelID 公開チャンネルまたはスレッドチャンネルのUUIDである
var IsPublicOrThreadChannelID = channelIDRule(func(ctx context.Context, cm channel.Manager, id uuid.UUID) (bool, error) {
	if cm.IsPublicChannel(ctx, id) {
		return true, nil
	}
	return cm.IsThreadChannel(ctx, id)
})

func channelIDRule(check func(context.Context, channel.Manager, uuid.UUID) (bool, error)) vd.Rule {
	return vd.WithContext(func(ctx context.Context, value any) error {
		const errMessage = "invalid channel id"

		cm, ok := ctx.Value(cmctxKey).(channel.Manager)
		if !ok {
			return vd.NewInternalError(errors.New("this context didn't have ChannelManager"))
		}

		var id uuid.UUID
		switch v := value.(type) {
		case nil:
			return nil
		case uuid.UUID:
			id = v
		case optional.Of[uuid.UUID]:
			if !v.Valid {
				return nil
			}
			id = v.V
		case string:
			id = uuid.FromStringOrNil(v)
		case []byte:
			id = uuid.FromBytesOrNil(v)
		default:
			return errors.New(errMessage)
		}

		valid, err := check(ctx, cm, id)
		if err != nil {
			if errors.Is(err, channel.ErrChannelNotFound) {
				return errors.New(errMessage)
			}
			return vd.NewInternalError(err)
		}
		if !valid {
			return errors.New(errMessage)
		}
		return nil
	})
}

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
		u, err = repo.GetUser(ctx, v, false)
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		u, err = repo.GetUser(ctx, v.V, false)
	case string:
		u, err = repo.GetUser(ctx, uuid.FromStringOrNil(v), false)
	case []byte:
		u, err = repo.GetUser(ctx, uuid.FromBytesOrNil(v), false)
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
		ok, err = repo.UserExists(ctx, v)
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		ok, err = repo.UserExists(ctx, v.V)
	case string:
		ok, err = repo.UserExists(ctx, uuid.FromStringOrNil(v))
	case []byte:
		ok, err = repo.UserExists(ctx, uuid.FromBytesOrNil(v))
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
		user, err = repo.GetUser(ctx, v, false)
	case optional.Of[uuid.UUID]:
		if !v.Valid {
			return nil
		}
		user, err = repo.GetUser(ctx, v.V, false)
	case string:
		user, err = repo.GetUser(ctx, uuid.FromStringOrNil(v), false)
	case []byte:
		user, err = repo.GetUser(ctx, uuid.FromBytesOrNil(v), false)
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
