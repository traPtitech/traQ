package router

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sse"
	"github.com/traPtitech/traQ/utils/imagemagick"
	"github.com/traPtitech/traQ/webrtc"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
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
	paramBotID       = "botID"
	paramClientID    = "clientID"

	loggerKey  = "logger"
	traceIDKey = "traceId"
	userIDKey  = "userID"

	mimeImagePNG  = "image/png"
	mimeImageJPEG = "image/jpeg"
	mimeImageGIF  = "image/gif"
	mimeImageSVG  = "image/svg+xml"

	headerCacheControl      = "Cache-Control"
	headerETag              = "ETag"
	headerIfMatch           = "If-Match"
	headerIfNoneMatch       = "If-None-Match"
	headerIfModifiedSince   = "If-Modified-Since"
	headerIfUnmodifiedSince = "If-Unmodified-Since"
	headerFileMetaType      = "X-TRAQ-FILE-TYPE"
	headerCacheFile         = "X-TRAQ-FILE-CACHE"
	headerSignature         = "X-TRAQ-Signature"
	headerChannelID         = "X-TRAQ-Channel-Id"
	headerMore              = "X-TRAQ-More"

	unexpectedError = "unexpected error"
)

var json = jsoniter.ConfigFastest

func init() {
	gob.Register(uuid.UUID{})
}

type timestamp time.Time

// UnmarshalParam implements BindUnmarshaler interface.
func (t *timestamp) UnmarshalParam(src string) error {
	ts, err := time.Parse(time.RFC3339, src)
	*t = timestamp(ts)
	return err
}

// Handlers ハンドラ
type Handlers struct {
	RBAC     rbac.RBAC
	Repo     repository.Repository
	SSE      *sse.Streamer
	Hub      *hub.Hub
	Logger   *zap.Logger
	WebRTC   *webrtc.Manager
	Realtime *realtime.Manager
	HandlerConfig

	emojiJSONCache     bytes.Buffer
	emojiJSONTime      time.Time
	emojiJSONCacheLock sync.RWMutex
	emojiCSSCache      bytes.Buffer
	emojiCSSTime       time.Time
	emojiCSSCacheLock  sync.RWMutex

	messagesResponseCacheGroup singleflight.Group
}

// HandlerConfig ハンドラ設定
type HandlerConfig struct {
	// ImageMagickPath ImageMagickの実行パス
	ImageMagickPath string
	// AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	// IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool
	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string
}

// NewHandlers ハンドラを生成します
func NewHandlers(rbac rbac.RBAC, repo repository.Repository, hub *hub.Hub, logger *zap.Logger, realtime *realtime.Manager, config HandlerConfig) *Handlers {
	h := &Handlers{
		RBAC:          rbac,
		Repo:          repo,
		SSE:           sse.NewStreamer(),
		Hub:           hub,
		Logger:        logger,
		WebRTC:        webrtc.NewManager(hub),
		Realtime:      realtime,
		HandlerConfig: config,
	}
	go h.stampEventSubscriber(hub.Subscribe(10, event.StampCreated, event.StampUpdated, event.StampDeleted))
	return h
}

func (h *Handlers) stampEventSubscriber(sub hub.Subscription) {
	for range sub.Receiver {
		h.emojiJSONCacheLock.Lock()
		h.emojiJSONCache.Reset()
		h.emojiJSONCacheLock.Unlock()

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

func (h *Handlers) processMultipartFormIconUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > iconFileMaxSize {
		return uuid.Nil, badRequest("too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeIcon, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case mimeImagePNG, mimeImageJPEG:
			return h.processStillImage(c, src, iconMaxWidth, iconMaxHeight)
		case mimeImageGIF:
			return h.processGifImage(c, h.ImageMagickPath, src, iconMaxWidth, iconMaxHeight)
		}
		return nil, "", badRequest("invalid image file")
	})
}

func (h *Handlers) processMultipartFormStampUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > stampFileMaxSize {
		return uuid.Nil, badRequest("too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeStamp, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case mimeImagePNG, mimeImageJPEG:
			return h.processStillImage(c, src, stampMaxWidth, stampMaxHeight)
		case mimeImageGIF:
			return h.processGifImage(c, h.ImageMagickPath, src, stampMaxWidth, stampMaxHeight)
		case mimeImageSVG:
			return h.processSVGImage(c, src)
		}
		return nil, "", badRequest("invalid image file")
	})
}

func (h *Handlers) processMultipartForm(c echo.Context, file *multipart.FileHeader, fType string, process func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error)) (uuid.UUID, error) {
	// ファイルタイプ確認・必要があればリサイズ
	src, err := file.Open()
	if err != nil {
		return uuid.Nil, internalServerError(err, h.requestContextLogger(c))
	}
	b, mime, err := process(c, file.Header.Get(echo.HeaderContentType), src)
	src.Close()
	if err != nil {
		return uuid.Nil, err
	}

	// ファイル保存
	f, err := h.Repo.SaveFile(file.Filename, b, int64(b.Len()), mime, fType, uuid.Nil)
	if err != nil {
		return uuid.Nil, internalServerError(err, h.requestContextLogger(c))
	}

	return f.ID, nil
}

func (h *Handlers) processStillImage(c echo.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	img, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", badRequest("bad image file")
	}

	if size := img.Bounds().Size(); size.X > maxWidth || size.Y > maxHeight {
		img = imaging.Fit(img, maxWidth, maxHeight, imaging.Linear)
	}

	// bytesに戻す
	var b bytes.Buffer
	_ = imaging.Encode(&b, img, imaging.PNG)
	return &b, mimeImagePNG, nil
}

func (h *Handlers) processGifImage(c echo.Context, imagemagickPath string, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10秒以内に終わらないファイルは無効
	defer cancel()

	b, err := imagemagick.ResizeAnimationGIF(ctx, imagemagickPath, src, maxWidth, maxHeight, false)
	if err != nil {
		switch err {
		case imagemagick.ErrUnavailable:
			// gifは一時的にサポートされていない
			return nil, "", badRequest("gif file is temporarily unsupported")
		case imagemagick.ErrUnsupportedType:
			// 不正なgifである
			return nil, "", badRequest("bad image file")
		case context.DeadlineExceeded:
			// リサイズタイムアウト
			return nil, "", badRequest("bad image file (resize timeout)")
		default:
			// 予期しないエラー
			return nil, "", internalServerError(err, h.requestContextLogger(c))
		}
	}

	return b, mimeImageGIF, nil
}

func (h *Handlers) processSVGImage(c echo.Context, src io.Reader) (*bytes.Buffer, string, error) {
	// TODO svg検証
	b := &bytes.Buffer{}
	_, err := io.Copy(b, src)
	if err != nil {
		return nil, "", internalServerError(err, h.requestContextLogger(c))
	}
	return b, mimeImageSVG, nil
}

func (h *Handlers) getUserIcon(c echo.Context, user *model.User) error {
	// ファイルメタ取得
	meta, err := h.Repo.GetFileMeta(user.Icon)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	// ファイルオープン
	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(headerETag, strconv.Quote(meta.Hash))
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

func (h *Handlers) putUserIcon(c echo.Context, userID uuid.UUID) error {
	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return badRequest(err)
	}

	iconID, err := h.processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := h.Repo.ChangeUserIcon(userID, iconID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
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

// getTraceID トレースIDを返します
func getTraceID(c echo.Context) string {
	v, ok := c.Get(traceIDKey).(string)
	if ok {
		return v
	}
	v = fmt.Sprintf("%02x", uuid.Must(uuid.NewV4()).Bytes())
	c.Set(traceIDKey, v)
	return v
}

func (h *Handlers) requestContextLogger(c echo.Context) *zap.Logger {
	l, ok := c.Get(loggerKey).(*zap.Logger)
	if ok {
		return l
	}
	l = h.Logger.With(zap.String("logging.googleapis.com/trace", getTraceID(c)))
	c.Set(loggerKey, l)
	return l
}
