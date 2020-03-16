package v3

import (
	"bytes"
	"context"
	"github.com/disintegration/imaging"
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/imagemagick"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
	"time"
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
func getParamFile(c echo.Context) *model.File {
	return c.Get(consts.KeyParamFile).(*model.File)
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

// serveUserIcon userのアイコン画像ファイルをレスポンスとして返す
func serveUserIcon(c echo.Context, repo repository.Repository, user model.UserInfo) error {
	// ファイルメタ取得
	meta, err := repo.GetFileMeta(user.GetIconFileID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ファイルオープン
	file, err := repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	// レスポンスヘッダ設定
	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))

	// ファイル送信
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

// changeUserIcon userIDのユーザーのアイコン画像を変更するハンドラ
func changeUserIcon(c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	iconID, err := saveUploadImage(c, repo, "file", model.FileTypeIcon, 2<<20, 256)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := repo.ChangeUserIcon(userID, iconID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// saveUploadImage MultipartFormでアップロードされた画像ファイルをリサイズして保存
func saveUploadImage(c echo.Context, repo repository.Repository, name string, fType string, maxFileSize int64, maxImageSize int) (uuid.UUID, error) {
	// ファイルオープン
	src, file, err := c.Request().FormFile(name)
	if err != nil {
		return uuid.Nil, herror.BadRequest(err)
	}
	defer src.Close()

	// ファイルサイズ制限
	if file.Size > maxFileSize {
		return uuid.Nil, herror.BadRequest("too large image file (limit exceeded)")
	}

	// ファイルタイプ確認・必要があればリサイズ
	var (
		b    *bytes.Buffer
		mime string
	)
	switch file.Header.Get(echo.HeaderContentType) {
	case consts.MimeImagePNG, consts.MimeImageJPEG:
		// デコード
		img, err := imaging.Decode(src, imaging.AutoOrientation(true))
		if err != nil {
			return uuid.Nil, herror.BadRequest("bad image file")
		}

		// リサイズ
		if size := img.Bounds().Size(); size.X > maxImageSize || size.Y > maxImageSize {
			img = imaging.Fit(img, maxImageSize, maxImageSize, imaging.Linear)
		}

		// PNGに戻す
		b = &bytes.Buffer{}
		_ = imaging.Encode(b, img, imaging.PNG)
		mime = consts.MimeImagePNG
	case consts.MimeImageGIF:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10秒以内に終わらないファイルは無効
		defer cancel()

		// リサイズ
		b, err = imagemagick.ResizeAnimationGIF(ctx, imagemagickPath, src, maxImageSize, maxImageSize, false)
		if err != nil {
			switch err {
			case imagemagick.ErrUnavailable:
				// gifは一時的にサポートされていない
				return uuid.Nil, herror.BadRequest("gif file is temporarily unsupported")
			case imagemagick.ErrUnsupportedType:
				// 不正なgifである
				return uuid.Nil, herror.BadRequest("bad image file")
			case context.DeadlineExceeded:
				// リサイズタイムアウト
				return uuid.Nil, herror.BadRequest("bad image file (resize timeout)")
			default:
				// 予期しないエラー
				return uuid.Nil, herror.InternalServerError(err)
			}
		}
		mime = consts.MimeImageGIF
	default:
		return uuid.Nil, herror.BadRequest("invalid image file")
	}

	// ファイル保存
	f, err := repo.SaveFile(repository.SaveFileArgs{
		FileName: file.Filename,
		FileSize: int64(b.Len()),
		MimeType: mime,
		FileType: fType,
		Src:      b,
	})
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}

	return f.ID, nil
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
