package router

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/go-sql-driver/mysql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/external/imagemagick"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/utils/thumb"
	"image"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/labstack/echo"
)

const (
	iconMaxWidth    = 256
	iconMaxHeight   = 256
	iconFileMaxSize = 2 << 20

	stampMaxWidth    = 128
	stampMaxHeight   = 128
	stampFileMaxSize = 2 << 20

	errMySQLDuplicatedRecord uint16 = 1062

	paramChannelID   = "channelID"
	paramPinID       = "pinID"
	paramUserID      = "userID"
	paramTagID       = "tagID"
	paramStampID     = "stampID"
	paramMessageID   = "messageID"
	paramReferenceID = "referenceID"
	paramFileID      = "fileID"

	mimeImagePNG  = "image/png"
	mimeImageJPEG = "image/jpeg"
	mimeImageGIF  = "image/gif"
	mimeImageSVG  = "image/svg+xml"

	headerCacheControl = "Cache-Control"
)

// Handlers ハンドラ
type Handlers struct {
	Bot    *event.BotProcessor
	OAuth2 *oauth2.Handler
}

func init() {
	gob.Register(uuid.UUID{})
}

// CustomHTTPErrorHandler json形式でエラーレスポンスを返す
func CustomHTTPErrorHandler(err error, c echo.Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message
	} else {
		msg = http.StatusText(code)
	}
	if _, ok := msg.(string); ok {
		msg = map[string]interface{}{"message": msg}
	}

	if err = c.JSON(code, msg); err != nil {
		c.Echo().Logger.Errorf("an error occurred while sending to JSON: %v", err)
	}

}

func bindAndValidate(c echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := c.Validate(i); err != nil {
		return err
	}
	return nil
}

func isMySQLDuplicatedRecordErr(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLDuplicatedRecord
}

func processMultipartFormIconUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > iconFileMaxSize {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "too large image file (1MB limit)")
	}

	// ファイルタイプ確認・必要があればリサイズ
	src, err := file.Open()
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	b, err := processIcon(c, file.Header.Get(echo.HeaderContentType), src)
	src.Close()
	if err != nil {
		return uuid.Nil, err
	}

	// アイコン画像保存
	fileID, err := saveFile(file.Filename, b)
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}

	return fileID, nil
}

func processIcon(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, error) {
	switch mime {
	case mimeImagePNG, mimeImageJPEG:
		return processStillImage(c, src, iconMaxWidth, iconMaxHeight)
	case mimeImageGIF:
		return processGifImage(c, src, iconMaxWidth, iconMaxHeight)
	}
	return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
}

func processMultipartFormStampUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > stampFileMaxSize {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "too large image file (1MB limit)")
	}

	// ファイルタイプ確認・必要があればリサイズ
	src, err := file.Open()
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	b, err := processStamp(c, file.Header.Get(echo.HeaderContentType), src)
	src.Close()
	if err != nil {
		return uuid.Nil, err
	}

	// スタンプ画像保存
	fileID, err := saveFile(file.Filename, b)
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}

	return fileID, nil
}

func processStamp(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, error) {
	switch mime {
	case mimeImagePNG, mimeImageJPEG:
		return processStillImage(c, src, stampMaxWidth, stampMaxHeight)
	case mimeImageGIF:
		return processGifImage(c, src, stampMaxWidth, stampMaxHeight)
	case mimeImageSVG:
		return processSVGImage(c, src)
	}
	return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
}

func processStillImage(c echo.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "bad image file")
	}

	if img.Bounds().Size().X > maxWidth || img.Bounds().Size().Y > maxHeight {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
		defer cancel()
		img, err = thumb.Resize(ctx, img, maxWidth, maxHeight)
		if err != nil {
			switch err {
			case context.DeadlineExceeded:
				// リサイズタイムアウト
				return nil, echo.NewHTTPError(http.StatusBadRequest, "bad image file (resize timeout)")
			default:
				// 予期しないエラー
				c.Logger().Error(err)
				return nil, echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	// bytesに戻す
	b, err := thumb.EncodeToPNG(img)
	if err != nil {
		// 予期しないエラー
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}

	return b, nil
}

func processGifImage(c echo.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
	defer cancel()

	b, err := imagemagick.ResizeAnimationGIF(ctx, src, maxWidth, maxHeight, false)
	if err != nil {
		switch err {
		case imagemagick.ErrUnavailable:
			// gifは一時的にサポートされていない
			return nil, echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
		case imagemagick.ErrUnsupportedType:
			// 不正なgifである
			return nil, echo.NewHTTPError(http.StatusBadRequest, "bad image file")
		case context.DeadlineExceeded:
			// リサイズタイムアウト
			return nil, echo.NewHTTPError(http.StatusBadRequest, "bad image file (resize timeout)")
		default:
			// 予期しないエラー
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return b, nil
}

func processSVGImage(c echo.Context, src io.Reader) (*bytes.Buffer, error) {
	// TODO svg検証
	b := &bytes.Buffer{}
	_, err := io.Copy(b, src)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	return b, nil
}

func saveFile(name string, src *bytes.Buffer) (uuid.UUID, error) {
	file := &model.File{
		Name: name,
		Size: int64(src.Len()),
	}
	if err := file.Create(src); err != nil {
		return uuid.Nil, err
	}

	return file.GetID(), nil
}

func getRequestUser(c echo.Context) *model.User {
	return c.Get("user").(*model.User)
}

func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).GetUID()
}

func getRequestParamAsUUID(c echo.Context, name string) uuid.UUID {
	return uuid.FromStringOrNil(c.Param(name))
}

func getRBAC(c echo.Context) *rbac.RBAC {
	return c.Get("rbac").(*rbac.RBAC)
}
