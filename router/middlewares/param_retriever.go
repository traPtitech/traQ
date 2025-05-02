package middlewares

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
)

// ParamRetriever リクエストパスパラメータで指定された各種エンティティをrepositoryから取得するミドルウェア
type ParamRetriever struct {
	repo repository.Repository
	cm   channel.Manager
	mm   message.Manager
	fm   file.Manager
}

// NewParamRetriever ParamRetrieverを生成
func NewParamRetriever(repo repository.Repository, cm channel.Manager, fm file.Manager, mm message.Manager) *ParamRetriever {
	return &ParamRetriever{repo: repo, cm: cm, fm: fm, mm: mm}
}

func (pr *ParamRetriever) byString(param string, key string, f func(c echo.Context, v string) (interface{}, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r, err := f(c, c.Param(param))
			if err != nil {
				return pr.error(err)
			}

			c.Set(key, r)
			return next(c)
		}
	}
}

func (pr *ParamRetriever) byUUID(param string, key string, f func(c echo.Context, v uuid.UUID) (interface{}, error)) echo.MiddlewareFunc {
	return pr.byString(param, key, func(c echo.Context, v string) (interface{}, error) {
		u, err := uuid.FromString(v)
		if err != nil || u == uuid.Nil {
			return nil, herror.NotFound()
		}
		return f(c, u)
	})
}

func (pr *ParamRetriever) checkOnlyByUUID(param string, f func(c echo.Context, v uuid.UUID) (bool, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			v, err := uuid.FromString(c.Param(param))
			if err != nil {
				return herror.NotFound()
			}

			ok, err := f(c, v)
			if err != nil {
				return pr.error(err)
			}
			if !ok {
				return herror.NotFound()
			}

			return next(c)
		}
	}
}

func (pr *ParamRetriever) error(err error) error {
	switch err.(type) {
	case *echo.HTTPError:
		return err
	case *herror.InternalError:
		return err
	default:
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		case channel.ErrChannelNotFound:
			return herror.NotFound()
		case file.ErrNotFound:
			return herror.NotFound()
		case message.ErrNotFound:
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	}
}

// GroupID リクエストURLの`groupID`パラメータからGroupを取り出す
func (pr *ParamRetriever) GroupID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamGroupID, consts.KeyParamGroup, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetUserGroup(v)
	})
}

// MessageID リクエストURLの`messageID`パラメータからMessageを取り出す
func (pr *ParamRetriever) MessageID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamMessageID, consts.KeyParamMessage, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.mm.Get(v)
	})
}

// ClientID リクエストURLの`clientID`パラメータからOAuth2Clientを取り出す
func (pr *ParamRetriever) ClientID() echo.MiddlewareFunc {
	return pr.byString(consts.ParamClientID, consts.KeyParamClient, func(_ echo.Context, v string) (interface{}, error) {
		return pr.repo.GetClient(v)
	})
}

// BotID リクエストURLの`botID`パラメータからBotを取り出す
func (pr *ParamRetriever) BotID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamBotID, consts.KeyParamBot, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetBotByID(v)
	})
}

// ChannelID リクエストURLの`channelID`パラメータからChannelを取り出す
func (pr *ParamRetriever) ChannelID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamChannelID, consts.KeyParamChannel, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.cm.GetChannel(v)
	})
}

// FileID リクエストURLの`fileID`パラメータからFileを取り出す
func (pr *ParamRetriever) FileID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamFileID, consts.KeyParamFile, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.fm.Get(v)
	})
}

// WebhookID リクエストURLの`webhookID`パラメータからBotを取り出す
func (pr *ParamRetriever) WebhookID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamWebhookID, consts.KeyParamWebhook, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetWebhook(v)
	})
}

// StampID リクエストURLの`stampID`パラメータからStampを取り出します
func (pr *ParamRetriever) StampID(checkOnly bool) echo.MiddlewareFunc {
	if checkOnly {
		return pr.checkOnlyByUUID(consts.ParamStampID, func(_ echo.Context, v uuid.UUID) (bool, error) {
			return pr.repo.StampExists(v)
		})
	}
	return pr.byUUID(consts.ParamStampID, consts.KeyParamStamp, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetStamp(v)
	})
}

// StampPalettesID リクエストURLの`paletteID`パラメータからStampPaletteを取り出す
func (pr *ParamRetriever) StampPalettesID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamStampPaletteID, consts.KeyParamStampPalette, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetStampPalette(v)
	})
}

// UserID リクエストURLの`userID`パラメータからUserを取り出す
func (pr *ParamRetriever) UserID(checkOnly bool) echo.MiddlewareFunc {
	if checkOnly {
		return pr.checkOnlyByUUID(consts.ParamUserID, func(_ echo.Context, v uuid.UUID) (bool, error) {
			return pr.repo.UserExists(v)
		})
	}
	return pr.byUUID(consts.ParamUserID, consts.KeyParamUser, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetUser(v, true)
	})
}

// ClipFolderID リクエストURLの`folderID`パラメータからClipFolderを取り出す
func (pr *ParamRetriever) ClipFolderID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamClipFolderID, consts.KeyParamClipFolder, func(_ echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetClipFolder(v)
	})
}
