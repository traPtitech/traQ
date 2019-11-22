package v1

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/disintegration/imaging"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/middlewares"
	"github.com/traPtitech/traQ/router/sse"
	"github.com/traPtitech/traQ/utils/imagemagick"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	iconMaxWidth    = 256
	iconMaxHeight   = 256
	iconFileMaxSize = 2 << 20

	stampMaxWidth    = 128
	stampMaxHeight   = 128
	stampFileMaxSize = 2 << 20

	unexpectedError = "unexpected error"
)

var json = jsoniter.ConfigFastest

func init() {
	gob.Register(uuid.UUID{})
}

// Handlers ハンドラ
type Handlers struct {
	RBAC     rbac.RBAC
	Repo     repository.Repository
	SSE      *sse.Streamer
	WS       *ws.Streamer
	Hub      *hub.Hub
	Logger   *zap.Logger
	Realtime *realtime.Service

	// ImageMagickPath ImageMagickの実行パス
	ImageMagickPath string
	// AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	// IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool
	// SkyWaySecretKey SkyWayクレデンシャル用シークレットキー
	SkyWaySecretKey string

	webhookDefTmpls *template.Template

	emojiJSONCache     bytes.Buffer
	emojiJSONTime      time.Time
	emojiJSONCacheLock sync.RWMutex
	emojiCSSCache      bytes.Buffer
	emojiCSSTime       time.Time
	emojiCSSCacheLock  sync.RWMutex

	messagesResponseCacheGroup  singleflight.Group
	getStampsResponseCacheGroup singleflight.Group
	getUsersResponseCacheGroup  singleflight.Group
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	bodyLimit := middlewares.RequestBodyLengthLimit
	adminOnly := middlewares.AdminOnly
	botGuard := h.BotGuard

	api := e.Group("/1.0", middlewares.UserAuthenticate(h.Repo))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", h.GetUsers, requires(permission.GetUser))
			apiUsers.POST("", h.PostUsers, requires(permission.RegisterUser))
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", h.GetMe, requires(permission.GetMe))
				apiUsersMe.PATCH("", h.PatchMe, requires(permission.EditMe))
				apiUsersMe.PUT("/password", h.PutPassword, requires(permission.ChangeMyPassword))
				apiUsersMe.GET("/qr-code", h.GetMyQRCode, requires(permission.GetUserQRCode))
				apiUsersMe.GET("/icon", h.GetMyIcon, requires(permission.DownloadFile))
				apiUsersMe.PUT("/icon", h.PutMyIcon, requires(permission.ChangeMyIcon))
				apiUsersMe.GET("/stamp-history", h.GetMyStampHistory, requires(permission.GetMyStampHistory))
				apiUsersMe.GET("/groups", h.GetMyBelongingGroup, requires(permission.GetUserGroup))
				apiUsersMe.GET("/notification", h.GetMyNotificationChannels, requires(permission.GetChannelSubscription))
				apiUsersMe.GET("/tokens", h.GetMyTokens, requires(permission.GetMyTokens))
				apiUsersMe.DELETE("/tokens/:tokenID", h.DeleteMyToken, requires(permission.RevokeMyToken))
				apiUsersMeSessions := apiUsersMe.Group("/sessions")
				{
					apiUsersMeSessions.GET("", h.GetMySessions, requires(permission.GetMySessions))
					apiUsersMeSessions.DELETE("", h.DeleteAllMySessions, requires(permission.DeleteMySessions))
					apiUsersMeSessions.DELETE("/:referenceID", h.DeleteMySession, requires(permission.DeleteMySessions))
				}
				apiUsersMeStars := apiUsersMe.Group("/stars")
				{
					apiUsersMeStars.GET("", h.GetStars, requires(permission.GetChannelStar))
					apiUsersMeStarsCid := apiUsersMeStars.Group("/:channelID", h.ValidateChannelID(true))
					{
						apiUsersMeStarsCid.PUT("", h.PutStars, requires(permission.EditChannelStar))
						apiUsersMeStarsCid.DELETE("", h.DeleteStars, requires(permission.EditChannelStar))
					}
				}
				apiUsersMeUnread := apiUsersMe.Group("/unread")
				{
					apiUsersMeUnread.GET("/channels", h.GetUnreadChannels, requires(permission.GetUnread))
					apiUsersMeUnread.DELETE("/channels/:channelID", h.DeleteUnread, requires(permission.DeleteUnread))
				}
			}
			apiUsersUID := apiUsers.Group("/:userID", h.ValidateUserID(false))
			{
				apiUsersUID.GET("", h.GetUserByID, requires(permission.GetUser))
				apiUsersUID.PATCH("", h.PatchUserByID, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/status", h.PutUserStatus, requires(permission.EditOtherUsers))
				apiUsersUID.PUT("/password", h.PutUserPassword, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/messages", h.GetDirectMessages, requires(permission.GetMessage), botGuard(blockUnlessSubscribingEvent(model.BotEventDirectMessageCreated)))
				apiUsersUID.POST("/messages", h.PostDirectMessage, bodyLimit(100), requires(permission.PostMessage), botGuard(blockUnlessSubscribingEvent(model.BotEventDirectMessageCreated)))
				apiUsersUID.GET("/icon", h.GetUserIcon, requires(permission.DownloadFile))
				apiUsersUID.PUT("/icon", h.PutUserIcon, requires(permission.EditOtherUsers))
				apiUsersUID.GET("/notification", h.GetNotificationChannels, requires(permission.GetChannelSubscription))
				apiUsersUID.GET("/groups", h.GetUserBelongingGroup, requires(permission.GetUserGroup))
				apiUsersUIDTags := apiUsersUID.Group("/tags")
				{
					apiUsersUIDTags.GET("", h.GetUserTags, requires(permission.GetUserTag))
					apiUsersUIDTags.POST("", h.PostUserTag, requires(permission.EditUserTag))
					apiUsersUIDTagsTid := apiUsersUIDTags.Group("/:tagID")
					{
						apiUsersUIDTagsTid.PATCH("", h.PatchUserTag, requires(permission.EditUserTag))
						apiUsersUIDTagsTid.DELETE("", h.DeleteUserTag, requires(permission.EditUserTag))
					}
				}
			}
		}
		apiHeartBeat := api.Group("/heartbeat")
		{
			apiHeartBeat.GET("", h.GetHeartbeat, requires(permission.GetHeartbeat)) // Deprecated
			apiHeartBeat.POST("", h.PostHeartbeat, requires(permission.PostHeartbeat))
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", h.GetChannels, requires(permission.GetChannel))
			apiChannels.POST("", h.PostChannels, requires(permission.CreateChannel))
			apiChannelsCid := apiChannels.Group("/:channelID", h.ValidateChannelID(false))
			{
				apiChannelsCid.GET("", h.GetChannelByChannelID, requires(permission.GetChannel))
				apiChannelsCid.PATCH("", h.PatchChannelByChannelID, requires(permission.EditChannel))
				apiChannelsCid.DELETE("", h.DeleteChannelByChannelID, requires(permission.DeleteChannel))
				apiChannelsCid.PUT("/parent", h.PutChannelParent, requires(permission.ChangeParentChannel))
				apiChannelsCid.POST("/children", h.PostChannelChildren, requires(permission.CreateChannel))
				apiChannelsCid.GET("/pins", h.GetChannelPin, requires(permission.GetMessage))
				apiChannelsCid.GET("/events", h.GetChannelEvents, requires(permission.GetChannel))
				apiChannelsCid.GET("/stats", h.GetChannelStats, requires(permission.GetChannel))
				apiChannelsCid.GET("/viewers", h.GetChannelViewers, requires(permission.GetChannel))
				apiChannelsCidTopic := apiChannelsCid.Group("/topic")
				{
					apiChannelsCidTopic.GET("", h.GetTopic, requires(permission.GetChannel))
					apiChannelsCidTopic.PUT("", h.PutTopic, requires(permission.EditChannelTopic))
				}
				apiChannelsCidMessages := apiChannelsCid.Group("/messages")
				{
					apiChannelsCidMessages.GET("", h.GetMessagesByChannelID, requires(permission.GetMessage))
					apiChannelsCidMessages.POST("", h.PostMessage, bodyLimit(100), requires(permission.PostMessage))
				}
				apiChannelsCidNotification := apiChannelsCid.Group("/notification")
				{
					apiChannelsCidNotification.GET("", h.GetChannelSubscribers, requires(permission.GetChannelSubscription))
					apiChannelsCidNotification.PUT("", h.PutChannelSubscribers, requires(permission.EditChannelSubscription))
				}
				apiChannelsCidBots := apiChannelsCid.Group("/bots")
				{
					apiChannelsCidBots.GET("", h.GetChannelBots, requires(permission.GetBot))
					apiChannelsCidBots.POST("", h.PostChannelBots, requires(permission.InstallBot))
					apiChannelsCidBots.DELETE("/:botID", h.DeleteChannelBot, requires(permission.UninstallBot), h.ValidateBotID(false))
				}
				apiChannelsCidWebRTC := apiChannelsCid.Group("/webrtc")
				{
					apiChannelsCidWebRTC.GET("/state", h.GetChannelWebRTCState, requires(permission.GetChannel))
				}
			}
		}
		apiNotification := api.Group("/notification")
		{
			apiNotification.GET("", echo.WrapHandler(h.SSE), requires(permission.ConnectNotificationStream))
			apiNotification.POST("/device", h.PostDeviceToken, requires(permission.RegisterFCMDevice))
		}
		apiMessages := api.Group("/messages")
		{
			apiMessages.GET("/reports", h.GetMessageReports, requires(permission.GetMessageReports))
			apiMessagesMid := apiMessages.Group("/:messageID", h.ValidateMessageID())
			{
				apiMessagesMid.GET("", h.GetMessageByID, requires(permission.GetMessage))
				apiMessagesMid.PUT("", h.PutMessageByID, bodyLimit(100), requires(permission.EditMessage))
				apiMessagesMid.DELETE("", h.DeleteMessageByID, requires(permission.DeleteMessage))
				apiMessagesMid.POST("/report", h.PostMessageReport, requires(permission.ReportMessage))
				apiMessagesMid.GET("/stamps", h.GetMessageStamps, requires(permission.GetMessage))
				apiMessagesMidStampsSid := apiMessagesMid.Group("/stamps/:stampID", h.ValidateStampID(true))
				{
					apiMessagesMidStampsSid.POST("", h.PostMessageStamp, requires(permission.AddMessageStamp))
					apiMessagesMidStampsSid.DELETE("", h.DeleteMessageStamp, requires(permission.RemoveMessageStamp))
				}
			}
		}
		apiTags := api.Group("/tags")
		{
			apiTagsTid := apiTags.Group("/:tagID")
			{
				apiTagsTid.GET("", h.GetUsersByTagID, requires(permission.GetUserTag))
			}
		}
		apiFiles := api.Group("/files")
		{
			apiFiles.POST("", h.PostFile, bodyLimit(30<<10), requires(permission.UploadFile))
			apiFilesFid := apiFiles.Group("/:fileID", h.ValidateFileID())
			{
				apiFilesFid.GET("", h.GetFileByID, requires(permission.DownloadFile))
				apiFilesFid.DELETE("", h.DeleteFileByID, requires(permission.DeleteFile))
				apiFilesFid.GET("/meta", h.GetMetaDataByFileID, requires(permission.DownloadFile))
				apiFilesFid.GET("/thumbnail", h.GetThumbnailByID, requires(permission.DownloadFile))
			}
		}
		apiPins := api.Group("/pins")
		{
			apiPins.POST("", h.PostPin, requires(permission.CreateMessagePin))
			apiPinsPid := apiPins.Group("/:pinID", h.ValidatePinID())
			{
				apiPinsPid.GET("", h.GetPin, requires(permission.GetMessage))
				apiPinsPid.DELETE("", h.DeletePin, requires(permission.DeleteMessagePin))
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", h.GetStamps, requires(permission.GetStamp))
			apiStamps.POST("", h.PostStamp, requires(permission.CreateStamp))
			apiStampsSid := apiStamps.Group("/:stampID", h.ValidateStampID(false))
			{
				apiStampsSid.GET("", h.GetStamp, requires(permission.GetStamp))
				apiStampsSid.PATCH("", h.PatchStamp, requires(permission.EditStamp))
				apiStampsSid.DELETE("", h.DeleteStamp, requires(permission.DeleteStamp))
			}
		}
		apiWebhooks := api.Group("/webhooks")
		{
			apiWebhooks.GET("", h.GetWebhooks, requires(permission.GetWebhook))
			apiWebhooks.POST("", h.PostWebhooks, requires(permission.CreateWebhook))
			apiWebhooksWid := apiWebhooks.Group("/:webhookID", h.ValidateWebhookID(true))
			{
				apiWebhooksWid.GET("", h.GetWebhook, requires(permission.GetWebhook))
				apiWebhooksWid.PATCH("", h.PatchWebhook, requires(permission.EditWebhook))
				apiWebhooksWid.DELETE("", h.DeleteWebhook, requires(permission.DeleteWebhook))
				apiWebhooksWid.GET("/icon", h.GetWebhookIcon, requires(permission.GetWebhook))
				apiWebhooksWid.PUT("/icon", h.PutWebhookIcon, requires(permission.EditWebhook))
				apiWebhooksWid.GET("/messages", h.GetWebhookMessages, requires(permission.GetWebhook))
			}
		}
		apiGroups := api.Group("/groups")
		{
			apiGroups.GET("", h.GetUserGroups, requires(permission.GetUserGroup))
			apiGroups.POST("", h.PostUserGroups, requires(permission.CreateUserGroup))
			apiGroupsGid := apiGroups.Group("/:groupID", h.ValidateGroupID())
			{
				apiGroupsGid.GET("", h.GetUserGroup, requires(permission.GetUserGroup))
				apiGroupsGid.PATCH("", h.PatchUserGroup, requires(permission.EditUserGroup))
				apiGroupsGid.DELETE("", h.DeleteUserGroup, requires(permission.DeleteUserGroup))
				apiGroupsGidMembers := apiGroupsGid.Group("/members")
				{
					apiGroupsGidMembers.GET("", h.GetUserGroupMembers, requires(permission.GetUserGroup))
					apiGroupsGidMembers.POST("", h.PostUserGroupMembers, requires(permission.EditUserGroup))
					apiGroupsGidMembers.DELETE("/:userID", h.DeleteUserGroupMembers, requires(permission.EditUserGroup))
				}
			}
		}
		apiClients := api.Group("/clients")
		{
			apiClients.GET("", h.GetClients, requires(permission.GetClients))
			apiClients.POST("", h.PostClients, requires(permission.CreateClient))
			apiClientCid := apiClients.Group("/:clientID")
			{
				apiClientCid.GET("", h.GetClient, requires(permission.GetClients), h.ValidateClientID(false))
				apiClientCid.PATCH("", h.PatchClient, requires(permission.EditMyClient), h.ValidateClientID(true))
				apiClientCid.DELETE("", h.DeleteClient, requires(permission.DeleteMyClient), h.ValidateClientID(true))
				apiClientCid.GET("/detail", h.GetClientDetail, requires(permission.GetClients), h.ValidateClientID(true))
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", h.GetBots, requires(permission.GetBot))
			apiBots.POST("", h.PostBots, requires(permission.CreateBot))
			apiBotsBid := apiBots.Group("/:botID")
			{
				apiBotsBid.GET("", h.GetBot, requires(permission.GetBot), h.ValidateBotID(false))
				apiBotsBid.PATCH("", h.PatchBot, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.DELETE("", h.DeleteBot, requires(permission.DeleteBot), h.ValidateBotID(true))
				apiBotsBid.GET("/detail", h.GetBotDetail, requires(permission.GetBot), h.ValidateBotID(true))
				apiBotsBid.PUT("/events", h.PutBotEvents, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.GET(`/events/logs`, h.GetBotEventLogs, requires(permission.GetBot), h.ValidateBotID(true))
				apiBotsBid.GET("/icon", h.GetBotIcon, requires(permission.GetBot), h.ValidateBotID(false))
				apiBotsBid.PUT("/icon", h.PutBotIcon, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.PUT("/state", h.PutBotState, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.POST("/reissue", h.PostBotReissueTokens, requires(permission.EditBot), h.ValidateBotID(true))
				apiBotsBid.GET("/channels", h.GetBotJoinChannels, requires(permission.GetBot), h.ValidateBotID(true))
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/latest-messages", h.GetActivityLatestMessages, requires(permission.GetMessage))
		}
		apiAuthority := api.Group("/authority", adminOnly)
		{
			apiAuthorityRoles := apiAuthority.Group("/roles")
			{
				apiAuthorityRoles.GET("", h.GetRoles)
				apiAuthorityRoles.POST("", h.PostRoles)
				apiAuthorityRolesRid := apiAuthorityRoles.Group("/:role")
				{
					apiAuthorityRolesRid.GET("", h.GetRole)
					apiAuthorityRolesRid.PATCH("", h.PatchRole)
				}
			}
			apiAuthority.GET("/permissions", h.GetPermissions)
			apiAuthority.GET("/reload", h.GetAuthorityReload)
			apiAuthority.POST("/reload", h.PostAuthorityReload)
		}
		apiWebRTC := api.Group("/webrtc")
		{
			apiWebRTC.GET("/state", h.GetWebRTCState)
			apiWebRTC.PUT("/state", h.PutWebRTCState)
		}
		api.POST("/oauth2/authorize/decide", h.AuthorizationDecideHandler, botGuard(blockAlways))
		api.GET("/ws", echo.WrapHandler(h.WS), requires(permission.ConnectNotificationStream))

		if len(h.SkyWaySecretKey) > 0 {
			api.POST("/skyway/authenticate", h.PostSkyWayAuthenticate, botGuard(blockAlways))
		}
	}

	apiNoAuth := e.Group("/1.0")
	{
		apiNoAuth.POST("/login", h.PostLogin)
		apiNoAuth.POST("/logout", h.PostLogout)
		apiPublic := apiNoAuth.Group("/public")
		{
			apiPublic.GET("/icon/:username", h.GetPublicUserIcon)
			apiPublic.GET("/emoji.json", h.GetPublicEmojiJSON)
			apiPublic.GET("/emoji.css", h.GetPublicEmojiCSS)
			apiPublic.GET("/emoji/:stampID", h.GetPublicEmojiImage, h.ValidateStampID(false))
		}
		apiNoAuth.POST("/webhooks/:webhookID", h.PostWebhook, h.ValidateWebhookID(false))
		apiNoAuth.POST("/webhooks/:webhookID/github", h.PostWebhookByGithub, h.ValidateWebhookID(false))
		apiOAuth := apiNoAuth.Group("/oauth2")
		{
			apiOAuth.GET("/authorize", h.AuthorizationEndpointHandler)
			apiOAuth.POST("/authorize", h.AuthorizationEndpointHandler)
			apiOAuth.POST("/token", h.TokenEndpointHandler)
			apiOAuth.POST("/revoke", h.RevokeTokenEndpointHandler)
		}
	}

	go h.stampEventSubscriber(h.Hub.Subscribe(10, event.StampCreated, event.StampUpdated, event.StampDeleted))
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

// LoadWebhookTemplate Webhookのテンプレートファイルを読み込みます
func (h *Handlers) LoadWebhookTemplate(pattern string) {
	h.webhookDefTmpls = template.Must(template.New("").Funcs(template.FuncMap{
		"replace": strings.Replace,
	}).ParseGlob(pattern))
}

func bindAndValidate(c echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := validation.Validate(i); err != nil {
		if e, ok := err.(validation.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}

func (h *Handlers) processMultipartFormIconUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > iconFileMaxSize {
		return uuid.Nil, herror.BadRequest("too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeIcon, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case consts.MimeImagePNG, consts.MimeImageJPEG:
			return h.processStillImage(c, src, iconMaxWidth, iconMaxHeight)
		case consts.MimeImageGIF:
			return h.processGifImage(c, h.ImageMagickPath, src, iconMaxWidth, iconMaxHeight)
		}
		return nil, "", herror.BadRequest("invalid image file")
	})
}

func (h *Handlers) processMultipartFormStampUpload(c echo.Context, file *multipart.FileHeader) (uuid.UUID, error) {
	// ファイルサイズ制限
	if file.Size > stampFileMaxSize {
		return uuid.Nil, herror.BadRequest("too large image file (limit exceeded)")
	}
	return h.processMultipartForm(c, file, model.FileTypeStamp, func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error) {
		switch mime {
		case consts.MimeImagePNG, consts.MimeImageJPEG:
			return h.processStillImage(c, src, stampMaxWidth, stampMaxHeight)
		case consts.MimeImageGIF:
			return h.processGifImage(c, h.ImageMagickPath, src, stampMaxWidth, stampMaxHeight)
		case consts.MimeImageSVG:
			return h.processSVGImage(c, src)
		}
		return nil, "", herror.BadRequest("invalid image file")
	})
}

func (h *Handlers) processMultipartForm(c echo.Context, file *multipart.FileHeader, fType string, process func(c echo.Context, mime string, src io.Reader) (*bytes.Buffer, string, error)) (uuid.UUID, error) {
	// ファイルタイプ確認・必要があればリサイズ
	src, err := file.Open()
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}
	b, mime, err := process(c, file.Header.Get(echo.HeaderContentType), src)
	src.Close()
	if err != nil {
		return uuid.Nil, err
	}

	// ファイル保存
	f, err := h.Repo.SaveFile(file.Filename, b, int64(b.Len()), mime, fType, uuid.Nil)
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}

	return f.ID, nil
}

func (h *Handlers) processStillImage(c echo.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	img, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", herror.BadRequest("bad image file")
	}

	if size := img.Bounds().Size(); size.X > maxWidth || size.Y > maxHeight {
		img = imaging.Fit(img, maxWidth, maxHeight, imaging.Linear)
	}

	// bytesに戻す
	var b bytes.Buffer
	_ = imaging.Encode(&b, img, imaging.PNG)
	return &b, consts.MimeImagePNG, nil
}

func (h *Handlers) processGifImage(c echo.Context, imagemagickPath string, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10秒以内に終わらないファイルは無効
	defer cancel()

	b, err := imagemagick.ResizeAnimationGIF(ctx, imagemagickPath, src, maxWidth, maxHeight, false)
	if err != nil {
		switch err {
		case imagemagick.ErrUnavailable:
			// gifは一時的にサポートされていない
			return nil, "", herror.BadRequest("gif file is temporarily unsupported")
		case imagemagick.ErrUnsupportedType:
			// 不正なgifである
			return nil, "", herror.BadRequest("bad image file")
		case context.DeadlineExceeded:
			// リサイズタイムアウト
			return nil, "", herror.BadRequest("bad image file (resize timeout)")
		default:
			// 予期しないエラー
			return nil, "", herror.InternalServerError(err)
		}
	}

	return b, consts.MimeImageGIF, nil
}

func (h *Handlers) processSVGImage(c echo.Context, src io.Reader) (*bytes.Buffer, string, error) {
	// TODO svg検証
	b := &bytes.Buffer{}
	_, err := io.Copy(b, src)
	if err != nil {
		return nil, "", herror.InternalServerError(err)
	}
	return b, consts.MimeImageSVG, nil
}

func (h *Handlers) getUserIcon(c echo.Context, user *model.User) error {
	// ファイルメタ取得
	meta, err := h.Repo.GetFileMeta(user.Icon)
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ファイルオープン
	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

func (h *Handlers) putUserIcon(c echo.Context, userID uuid.UUID) error {
	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return herror.BadRequest(err)
	}

	iconID, err := h.processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := h.Repo.ChangeUserIcon(userID, iconID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func getRequestUser(c echo.Context) *model.User {
	return c.Get(consts.KeyUser).(*model.User)
}

func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).ID
}

func getRequestParamAsUUID(c echo.Context, name string) uuid.UUID {
	return extension.GetRequestParamAsUUID(c, name)
}

func (h *Handlers) requestContextLogger(c echo.Context) *zap.Logger {
	l, ok := c.Get(consts.KeyLogger).(*zap.Logger)
	if ok {
		return l
	}
	l = h.Logger.With(zap.String("logging.googleapis.com/trace", extension.GetTraceID(c)))
	c.Set(consts.KeyLogger, l)
	return l
}
