package middlewares

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"golang.org/x/sync/singleflight"
)

// ParamRetriever リクエストパスパラメータで指定された各種エンティティをrepositoryから取得するミドルウェア
type ParamRetriever struct {
	repo         repository.Repository
	messageCache singleflight.Group
}

// NewParamRetriever ParamRetrieverを生成
func NewParamRetriever(repo repository.Repository) *ParamRetriever {
	return &ParamRetriever{repo: repo}
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
		if err == repository.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	}
}

// GroupID リクエストURLの`groupID`パラメータからGroupを取り出す
func (pr *ParamRetriever) GroupID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamGroupID, consts.KeyParamGroup, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetUserGroup(v)
	})
}

// MessageID リクエストURLの`messageID`パラメータからMessageを取り出す
func (pr *ParamRetriever) MessageID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamMessageID, consts.KeyParamMessage, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		mI, err, _ := pr.messageCache.Do(v.String(), func() (interface{}, error) { return pr.repo.GetMessageByID(v) })
		return mI, err
	})
}

// ClientID リクエストURLの`clientID`パラメータからOAuth2Clientを取り出す
func (pr *ParamRetriever) ClientID() echo.MiddlewareFunc {
	return pr.byString(consts.ParamClientID, consts.KeyParamClient, func(c echo.Context, v string) (interface{}, error) {
		return pr.repo.GetClient(v)
	})
}

// BotID リクエストURLの`botID`パラメータからBotを取り出す
func (pr *ParamRetriever) BotID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamBotID, consts.KeyParamBot, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetBotByID(v)
	})
}

// ChannelID リクエストURLの`channelID`パラメータからChannelを取り出す
func (pr *ParamRetriever) ChannelID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamChannelID, consts.KeyParamChannel, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetChannel(v)
	})
}

// FileID リクエストURLの`fileID`パラメータからFileを取り出す
func (pr *ParamRetriever) FileID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamFileID, consts.KeyParamFile, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetFileMeta(v)
	})
}

// WebhookID リクエストURLの`webhookID`パラメータからBotを取り出す
func (pr *ParamRetriever) WebhookID() echo.MiddlewareFunc {
	return pr.byUUID(consts.ParamWebhookID, consts.KeyParamWebhook, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetWebhook(v)
	})
}

// StampID リクエストURLの`stampID`パラメータからStampを取り出します
func (pr *ParamRetriever) StampID(checkOnly bool) echo.MiddlewareFunc {
	if checkOnly {
		return pr.checkOnlyByUUID(consts.ParamStampID, func(c echo.Context, v uuid.UUID) (bool, error) {
			return pr.repo.StampExists(v)
		})
	}
	return pr.byUUID(consts.ParamStampID, consts.KeyParamStamp, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetStamp(v)
	})
}

// UserID リクエストURLの`userID`パラメータからUserを取り出す
func (pr *ParamRetriever) UserID(checkOnly bool) echo.MiddlewareFunc {
	if checkOnly {
		return pr.checkOnlyByUUID(consts.ParamUserID, func(c echo.Context, v uuid.UUID) (bool, error) {
			return pr.repo.UserExists(v)
		})
	}
	return pr.byUUID(consts.ParamUserID, consts.KeyParamUser, func(c echo.Context, v uuid.UUID) (interface{}, error) {
		return pr.repo.GetUser(v)
	})
}
