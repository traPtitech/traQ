package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
)

// NotImplemented 未実装API. 501 NotImplementedを返す
func NotImplemented(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented)
}

// bindAndValidate 構造体iにFormDataまたはJsonをデシリアライズします
func bindAndValidate(c echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := vd.Validate(i); err != nil {
		if e, ok := err.(vd.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}

// isTrue 文字列sが"1", "t", "T", "true", "TRUE", "True"の場合にtrueを返す
func isTrue(s string) (b bool) {
	b, _ = strconv.ParseBool(s)
	return
}

// getRequestUser リクエストしてきたユーザーの情報を取得
func getRequestUser(c echo.Context) model.UserInfo {
	return c.Get(consts.KeyUser).(model.UserInfo)
}

// getRequestUserID リクエストしてきたユーザーUUIDを取得
func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).GetID()
}

// getParamUser URLの:userIDに対応するユーザー構造体を取得
func getParamUser(c echo.Context) model.UserInfo {
	return c.Get(consts.KeyParamUser).(model.UserInfo)
}

// getParamWebhook URLの:webhookIDに対応するWebhookを取得
func getParamWebhook(c echo.Context) model.Webhook {
	return c.Get(consts.KeyParamWebhook).(model.Webhook)
}

// getParamBot URLの:botIDに対応するBotを取得
func getParamBot(c echo.Context) *model.Bot {
	return c.Get(consts.KeyParamBot).(*model.Bot)
}

// getParamClient URLの:clientIDに対応するOAuth2Clientを取得
func getParamClient(c echo.Context) *model.OAuth2Client {
	return c.Get(consts.KeyParamClient).(*model.OAuth2Client)
}

// getParamFile URLの:fileIDに対応するFileを取得
func getParamFile(c echo.Context) model.FileMeta {
	return c.Get(consts.KeyParamFile).(model.FileMeta)
}

// getParamStamp URLの:stampIDに対応するStampを取得
func getParamStamp(c echo.Context) *model.Stamp {
	return c.Get(consts.KeyParamStamp).(*model.Stamp)
}

// getParamChannel URLの:channelIDに対応するChannelを取得
func getParamChannel(c echo.Context) *model.Channel {
	return c.Get(consts.KeyParamChannel).(*model.Channel)
}

// getParamMessage URLの:messageIDに対応するMessageを取得
func getParamMessage(c echo.Context) *model.Message {
	return c.Get(consts.KeyParamMessage).(*model.Message)
}

// getParamGroup URLの:groupIDに対応するUserGroupを取得
func getParamGroup(c echo.Context) *model.UserGroup {
	return c.Get(consts.KeyParamGroup).(*model.UserGroup)
}

// getParamAsUUID URLのnameパラメータの文字列をuuid.UUIDとして取得
func getParamAsUUID(c echo.Context, name string) uuid.UUID {
	return extension.GetRequestParamAsUUID(c, name)
}

type MessagesQuery struct {
	Limit     int       `query:"limit"`
	Offset    int       `query:"offset"`
	Since     null.Time `query:"since"`
	Until     null.Time `query:"until"`
	Inclusive bool      `query:"inclusive"`
	Order     string    `query:"order"`
}

func (q *MessagesQuery) bind(c echo.Context) error {
	return bindAndValidate(c, q)
}

func (q *MessagesQuery) Validate() error {
	if q.Limit == 0 {
		q.Limit = 20
	}
	return vd.ValidateStruct(q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(200)),
		vd.Field(&q.Offset, vd.Min(0)),
	)
}

func (q *MessagesQuery) convert() repository.MessagesQuery {
	return repository.MessagesQuery{
		Since:     q.Since,
		Until:     q.Until,
		Inclusive: q.Inclusive,
		Limit:     q.Limit,
		Offset:    q.Offset,
		Asc:       strings.ToLower(q.Order) == "asc",
	}
}

func (q *MessagesQuery) convertC(cid uuid.UUID) repository.MessagesQuery {
	r := q.convert()
	r.Channel = cid
	return r
}

func (q *MessagesQuery) convertU(uid uuid.UUID) repository.MessagesQuery {
	r := q.convert()
	r.User = uid
	return r
}

func serveMessages(c echo.Context, repo repository.Repository, query repository.MessagesQuery) error {
	messages, more, err := repo.GetMessages(query)
	if err != nil {
		return herror.InternalServerError(err)
	}
	c.Response().Header().Set(consts.HeaderMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, formatMessages(messages))
}
