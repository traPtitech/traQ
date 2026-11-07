package v3

import (
	"errors"

	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
)

// getAccessibleMessage URLの:messageIDに対応するMessageを取得し、リクエストユーザーがアクセス可能かを確認する
//
// メッセージが存在しない場合と、メッセージのチャンネルにアクセスできない場合はどちらも404を返す。
func (h *Handlers) getAccessibleMessage(c *echo.Context) (message.Message, error) {
	ctx := c.Request().Context()

	id, err := parseUUIDParam(c, consts.ParamMessageID)
	if err != nil {
		return nil, err
	}

	m, err := h.MessageManager.Get(ctx, id)
	if err != nil {
		if errors.Is(err, message.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	ok, err := h.ChannelManager.IsChannelAccessibleToUser(ctx, getRequestUserID(c), m.GetChannelID())
	if err != nil {
		return nil, herror.InternalServerError(err)
	}
	if !ok {
		return nil, herror.NotFound()
	}

	return m, nil
}

// getAccessibleChannel URLの:channelIDに対応するChannelを取得し、リクエストユーザーがアクセス可能かを確認する
//
// チャンネルが存在しない場合と、アクセスできない場合はどちらも404を返す。
func (h *Handlers) getAccessibleChannel(c *echo.Context) (*model.Channel, error) {
	ctx := c.Request().Context()

	id, err := parseUUIDParam(c, consts.ParamChannelID)
	if err != nil {
		return nil, err
	}

	ch, err := h.ChannelManager.GetChannel(ctx, id)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	ok, err := h.ChannelManager.IsChannelAccessibleToUser(ctx, getRequestUserID(c), ch.ID)
	if err != nil {
		return nil, herror.InternalServerError(err)
	}
	if !ok {
		return nil, herror.NotFound()
	}

	return ch, nil
}
