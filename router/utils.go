package router

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/go-sql-driver/mysql"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/imagemagick"
	"github.com/traPtitech/traQ/utils/thumb"
	"image"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"io"
	"mime/multipart"
	"net/http"
	"sync"
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
	paramGroupID     = "groupID"
	paramTagID       = "tagID"
	paramStampID     = "stampID"
	paramMessageID   = "messageID"
	paramReferenceID = "referenceID"
	paramFileID      = "fileID"
	paramWebhookID   = "webhookID"
	paramClipID      = "clipID"
	paramFolderID    = "folderID"
	paramTokenID     = "tokenID"

	mimeImagePNG  = "image/png"
	mimeImageJPEG = "image/jpeg"
	mimeImageGIF  = "image/gif"
	mimeImageSVG  = "image/svg+xml"

	headerCacheControl = "Cache-Control"
	headerFileMetaType = "X-TRAQ-FILE-TYPE"
	headerCacheFile    = "X-TRAQ-FILE-CACHE"
	headerSignature    = "X-TRAQ-Signature"
	headerChannelID    = "X-TRAQ-Channel-Id"
)

func init() {
	gob.Register(uuid.UUID{})
}

// Handlers ハンドラ
type Handlers struct {
	OAuth2          *oauth2.Handler
	RBAC            *rbac.RBAC
	Repo            repository.Repository
	SSE             *SSEStreamer
	Hub             *hub.Hub
	ImageMagickPath string

	emojiJsonCache     bytes.Buffer
	emojiJsonCacheLock sync.RWMutex
	emojiCSSCache      bytes.Buffer
	emojiCSSCacheLock  sync.RWMutex
}

// NewHandlers ハンドラを生成します
func NewHandlers(oauth2 *oauth2.Handler, rbac *rbac.RBAC, repo repository.Repository, hub *hub.Hub, imageMagickPath string) *Handlers {
	h := &Handlers{
		OAuth2:          oauth2,
		RBAC:            rbac,
		Repo:            repo,
		SSE:             NewSSEStreamer(hub, repo),
		Hub:             hub,
		ImageMagickPath: imageMagickPath,
	}
	go h.stampEventSubscriber(hub.Subscribe(10, event.StampCreated, event.StampUpdated, event.StampDeleted))
	return h
}

func (h *Handlers) stampEventSubscriber(sub hub.Subscription) {
	for range sub.Receiver {
		h.emojiJsonCacheLock.Lock()
		h.emojiJsonCache.Reset()
		h.emojiJsonCacheLock.Unlock()

		h.emojiCSSCacheLock.Lock()
		h.emojiCSSCache.Reset()
		h.emojiCSSCacheLock.Unlock()
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

func (h *Handlers) processMultipartFormIconUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > iconFileMaxSize {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeIcon, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case mimeImagePNG, mimeImageJPEG:
			return processStillImage(c, src, iconMaxWidth, iconMaxHeight)
		case mimeImageGIF:
			return processGifImage(c, h.ImageMagickPath, src, iconMaxWidth, iconMaxHeight)
		}
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	})
}

func (h *Handlers) processMultipartFormStampUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > stampFileMaxSize {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeStamp, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case mimeImagePNG, mimeImageJPEG:
			return processStillImage(c, src, stampMaxWidth, stampMaxHeight)
		case mimeImageGIF:
			return processGifImage(c, h.ImageMagickPath, src, stampMaxWidth, stampMaxHeight)
		case mimeImageSVG:
			return processSVGImage(c, src)
		}
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	})
}

func (h *Handlers) processMultipartForm(c echo.Context, file *multipart.FileHeader, fType string, process func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error)) (uuid.UUID, error) {
	// ファイルタイプ確認・必要があればリサイズ
	src, err := file.Open()
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	b, mime, err := process(c, file.Header.Get(echo.HeaderContentType), src)
	src.Close()
	if err != nil {
		return uuid.Nil, err
	}

	// ファイル保存
	f, err := h.Repo.SaveFile(file.Filename, b, int64(b.Len()), mime, fType, uuid.Nil)
	if err != nil {
		c.Logger().Error(err)
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError)
	}

	return f.ID, nil
}

func processStillImage(c echo.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "bad image file")
	}

	if size := img.Bounds().Size(); size.X > maxWidth || size.Y > maxHeight {
		img = thumb.Resize(img, maxWidth, maxHeight)
	}

	// bytesに戻す
	b, err := thumb.EncodeToPNG(img)
	if err != nil {
		// 予期しないエラー
		c.Logger().Error(err)
		return nil, "", echo.NewHTTPError(http.StatusInternalServerError)
	}

	return b, mimeImagePNG, nil
}

func processGifImage(c echo.Context, imagemagickPath string, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
	defer cancel()

	b, err := imagemagick.ResizeAnimationGIF(ctx, imagemagickPath, src, maxWidth, maxHeight, false)
	if err != nil {
		switch err {
		case imagemagick.ErrUnavailable:
			// gifは一時的にサポートされていない
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
		case imagemagick.ErrUnsupportedType:
			// 不正なgifである
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "bad image file")
		case context.DeadlineExceeded:
			// リサイズタイムアウト
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "bad image file (resize timeout)")
		default:
			// 予期しないエラー
			c.Logger().Error(err)
			return nil, "", echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return b, mimeImageGIF, nil
}

func processSVGImage(c echo.Context, src io.Reader) (*bytes.Buffer, string, error) {
	// TODO svg検証
	b := &bytes.Buffer{}
	_, err := io.Copy(b, src)
	if err != nil {
		c.Logger().Error(err)
		return nil, "", echo.NewHTTPError(http.StatusInternalServerError)
	}
	return b, mimeImageSVG, nil
}

func getRequestUser(c echo.Context) *model.User {
	return c.Get("user").(*model.User)
}

func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).ID
}

func getRequestParamAsUUID(c echo.Context, name string) uuid.UUID {
	return uuid.FromStringOrNil(c.Param(name))
}

func getRBAC(c echo.Context) *rbac.RBAC {
	return c.Get("rbac").(*rbac.RBAC)
}
