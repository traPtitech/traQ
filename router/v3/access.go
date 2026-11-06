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
	id, err := parseUUIDParam(c, consts.ParamClipFolderID)
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

// getBot URLの:botIDに対応するBotを取得する
//
// アクセス権確認は行わない。Botが存在しない場合は404を返す。
func (h *Handlers) getBot(c *echo.Context) (*model.Bot, error) {
	id, err := parseUUIDParam(c, consts.ParamBotID)
	if err != nil {
		return nil, err
	}

	b, err := h.Repo.GetBotByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	return b, nil
}

// getAccessibleBot URLの:botIDに対応するBotを取得し、リクエストユーザーが管理できるかを確認する
//
// Botが存在しない場合は404を返す。
// Bot自身、Botの作成者、他人のBotへのアクセス権限を持つユーザーのみアクセスでき、それ以外は403を返す。
func (h *Handlers) getAccessibleBot(c *echo.Context) (*model.Bot, error) {
	b, err := h.getBot(c)
	if err != nil {
		return nil, err
	}

	user := getRequestUser(c)
	switch {
	case b.BotUserID == user.GetID(): // Bot自身のアクセス
	case b.CreatorID == user.GetID(): // Bot管理人のアクセス
	case h.RBAC.IsGranted(user.GetRole(), permission.AccessOthersBot): // 特権使用のアクセス
	default:
		return nil, herror.Forbidden()
	}

	return b, nil
}

// getUserGroup URLの:groupIDに対応するUserGroupを取得する
//
// アクセス権確認は行わない。グループが存在しない場合は404を返す。
func (h *Handlers) getUserGroup(c *echo.Context) (*model.UserGroup, error) {
	id, err := parseUUIDParam(c, consts.ParamGroupID)
	if err != nil {
		return nil, err
	}

	g, err := h.Repo.GetUserGroup(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	return g, nil
}

// getAdministrableUserGroup URLの:groupIDに対応するUserGroupを取得し、リクエストユーザーが管理できるかを確認する
//
// グループが存在しない場合は404を返す。
// グループの管理者、または全グループの管理権限を持つユーザーのみアクセスでき、それ以外は403を返す。
func (h *Handlers) getAdministrableUserGroup(c *echo.Context) (*model.UserGroup, error) {
	g, err := h.getUserGroup(c)
	if err != nil {
		return nil, err
	}

	user := getRequestUser(c)
	if !g.IsAdmin(user.GetID()) && !h.RBAC.IsGranted(user.GetRole(), permission.AllUserGroupsAdmin) {
		return nil, herror.Forbidden()
	}

	return g, nil
}

// getClient URLの:clientIDに対応するOAuth2Clientを取得する
//
// アクセス権確認は行わない。クライアントが存在しない場合は404を返す。
func (h *Handlers) getClient(c *echo.Context) (*model.OAuth2Client, error) {
	oc, err := h.Repo.GetClient(c.Request().Context(), c.Param(consts.ParamClientID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	return oc, nil
}

// getAccessibleClient URLの:clientIDに対応するOAuth2Clientを取得し、リクエストユーザーが管理できるかを確認する
//
// クライアントが存在しない場合は404を返す。
// 作成者、または他人のクライアントの管理権限を持つユーザーのみアクセスでき、それ以外は403を返す。
func (h *Handlers) getAccessibleClient(c *echo.Context) (*model.OAuth2Client, error) {
	oc, err := h.getClient(c)
	if err != nil {
		return nil, err
	}

	user := getRequestUser(c)
	if oc.CreatorID != user.GetID() && !h.RBAC.IsGranted(user.GetRole(), permission.ManageOthersClient) {
		return nil, herror.Forbidden()
	}

	return oc, nil
}

// getStamp URLの:stampIDに対応するStampを取得する
//
// アクセス権確認は行わない。スタンプが存在しない場合は404を返す。
func (h *Handlers) getStamp(c *echo.Context) (*model.Stamp, error) {
	id, err := parseUUIDParam(c, consts.ParamStampID)
	if err != nil {
		return nil, err
	}

	s, err := h.Repo.GetStamp(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	return s, nil
}

// getDeletableStamp URLの:stampIDに対応するStampを取得し、リクエストユーザーが削除できるかを確認する
//
// スタンプが存在しない場合は404を返す。
// スタンプ削除権限を持つユーザー、または自分のスタンプの削除権限を持つ作成者のみアクセスでき、それ以外は403を返す。
func (h *Handlers) getDeletableStamp(c *echo.Context) (*model.Stamp, error) {
	s, err := h.getStamp(c)
	if err != nil {
		return nil, err
	}

	user := getRequestUser(c)
	switch {
	case h.RBAC.IsGranted(user.GetRole(), permission.DeleteStamp): // adminのアクセス
	case h.RBAC.IsGranted(user.GetRole(), permission.DeleteMyStamp) && s.CreatorID == user.GetID(): // スタンプ作成者のアクセス
	default:
		return nil, herror.Forbidden()
	}

	return s, nil
}
