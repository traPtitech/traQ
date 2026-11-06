package v3

import (
	"errors"

	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac/permission"
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

// getAccessibleFile URLの:fileIDに対応するFileを取得し、リクエストユーザーがアクセス可能かを確認する
//
// ファイルが存在しない場合は404、アクセスできない場合は403を返す。
func (h *Handlers) getAccessibleFile(c *echo.Context) (model.File, error) {
	return extension.GetAccessibleFile(c, h.FileManager, getRequestUserID(c))
}

// getWebhook URLの:webhookIDに対応するWebhookを取得する
//
// アクセス権確認は行わない。認証を伴わないWebhookへの投稿でのみ使うこと。
// Webhookが存在しない場合は404を返す。
func (h *Handlers) getWebhook(c *echo.Context) (model.Webhook, error) {
	id, err := parseUUIDParam(c, consts.ParamWebhookID)
	if err != nil {
		return nil, err
	}

	w, err := h.Repo.GetWebhook(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	return w, nil
}

// getAccessibleWebhook URLの:webhookIDに対応するWebhookを取得し、リクエストユーザーがアクセス可能かを確認する
//
// Webhookが存在しない場合は404を返す。
// 作成者以外は、他人のWebhookへのアクセス権限を持つ場合のみアクセスでき、それ以外は403を返す。
func (h *Handlers) getAccessibleWebhook(c *echo.Context) (model.Webhook, error) {
	w, err := h.getWebhook(c)
	if err != nil {
		return nil, err
	}

	user := getRequestUser(c)
	if w.GetCreatorID() != user.GetID() && !h.RBAC.IsGranted(user.GetRole(), permission.AccessOthersWebhook) {
		return nil, herror.Forbidden()
	}

	return w, nil
}

// getAccessibleClipFolder URLの:folderIDに対応するClipFolderを取得し、リクエストユーザーがアクセス可能かを確認する
//
// クリップフォルダが存在しない場合は404、所有者以外のアクセスは403を返す。
func (h *Handlers) getAccessibleClipFolder(c *echo.Context) (*model.ClipFolder, error) {
	id, err := utils.ParseUUIDParam(c, consts.ParamClipFolderID)
	if err != nil {
		return nil, err
	}

	cf, err := h.Repo.GetClipFolder(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	if cf.OwnerID != getRequestUserID(c) {
		return nil, herror.Forbidden()
	}

	return cf, nil
}
