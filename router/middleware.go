package router

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/logging"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

// AccessLoggingMiddleware アクセスログミドルウェア
func AccessLoggingMiddleware(logger *zap.Logger, excludesHeartbeat bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if excludesHeartbeat && strings.HasPrefix(c.Path(), "/api/1.0/heartbeat") {
				return next(c)
			}

			start := time.Now()
			if err := next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()

			req := c.Request()
			res := c.Response()
			logger.Info("", zap.String("logging.googleapis.com/trace", getTraceID(c)), logging.HTTPRequest(&logging.HTTPPayload{
				RequestMethod: req.Method,
				Status:        res.Status,
				UserAgent:     req.UserAgent(),
				RemoteIP:      c.RealIP(),
				Referer:       req.Referer(),
				Protocol:      req.Proto,
				RequestURL:    req.URL.String(),
				RequestSize:   req.Header.Get(echo.HeaderContentLength),
				ResponseSize:  strconv.FormatInt(res.Size, 10),
				Latency:       strconv.FormatFloat(stop.Sub(start).Seconds(), 'f', 9, 64) + "s",
			}))
			return nil
		}
	}
}

// UserAuthenticate User認証するミドルウェア
func (h *Handlers) UserAuthenticate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var user *model.User
			ah := c.Request().Header.Get(echo.HeaderAuthorization)
			if len(ah) > 0 {
				// AuthorizationヘッダーがあるためOAuth2で検証

				// Authorizationスキーム検証
				l := len(authScheme)
				if !(len(ah) > l+1 && ah[:l] == authScheme) {
					return echo.NewHTTPError(http.StatusUnauthorized, "the Authorization Header's scheme is invalid")
				}

				// OAuth2 Token検証
				token, err := h.Repo.GetTokenByAccess(ah[l+1:])
				if err != nil {
					switch err {
					case repository.ErrNotFound:
						return echo.NewHTTPError(http.StatusUnauthorized, "the token is invalid")
					default:
						return internalServerError(err, h.requestContextLogger(c))
					}
				}

				// tokenの有効期限の検証
				if token.IsExpired() {
					return echo.NewHTTPError(http.StatusUnauthorized, "the token is expired")
				}

				// tokenの検証に成功。ユーザーを取得
				user, err = h.Repo.GetUser(token.UserID)
				if err != nil {
					switch err {
					case repository.ErrNotFound:
						return echo.NewHTTPError(http.StatusUnauthorized, "the user is not found")
					default:
						return internalServerError(err, h.requestContextLogger(c))
					}
				}

				// 認可に基づきRole生成
				var roles []gorbac.Role
				for _, v := range token.Scopes {
					if r, ok := list[v]; ok && r != nil {
						roles = append(roles, r)
					}
				}
				c.Set("role", role.NewCompositeRole(roles...))
			} else {
				// Authorizationヘッダーがないためセッションを確認する
				sess, err := sessions.Get(c.Response(), c.Request(), false)
				if err != nil {
					return internalServerError(err, h.requestContextLogger(c))
				}
				if sess == nil || sess.GetUserID() == uuid.Nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "You are not logged in")
				}

				user, err = h.Repo.GetUser(sess.GetUserID())
				if err != nil {
					switch err {
					case repository.ErrNotFound:
						return echo.NewHTTPError(http.StatusUnauthorized, "the user is not found")
					default:
						return internalServerError(err, h.requestContextLogger(c))
					}
				}
			}

			// ユーザーアカウント状態を確認
			switch user.Status {
			case model.UserAccountStatusDeactivated, model.UserAccountStatusSuspended:
				return forbidden("this account is currently suspended")
			case model.UserAccountStatusActive:
				break
			}

			c.Set("user", user)
			c.Set("userID", user.ID)
			return next(c)
		}
	}
}

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(rbac *rbac.RBAC) func(p ...gorbac.Permission) echo.MiddlewareFunc {
	return func(p ...gorbac.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// クライアント権限検証
				if role, ok := c.Get("role").(gorbac.Role); ok {
					for _, v := range p {
						if !role.Permit(v) {
							// NG
							return forbidden(fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}
				}

				// ユーザー権限検証
				user := c.Get("user").(*model.User)
				for _, v := range p {
					if !rbac.IsGranted(user.ID, user.Role, v) {
						// NG
						return forbidden(fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
					}
				}
				c.Set("rbac", rbac)

				return next(c) // OK
			}
		}
	}
}

// RequestBodyLengthLimit リクエストボディのContentLengthで制限をかけるミドルウェア
func RequestBodyLengthLimit(kb int64) echo.MiddlewareFunc {
	limit := kb << 10
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if l := c.Request().Header.Get(echo.HeaderContentLength); len(l) == 0 {
				return echo.NewHTTPError(http.StatusLengthRequired) // ContentLengthを送ってこないリクエストを殺す
			}
			if c.Request().ContentLength > limit {
				return echo.NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("the request must be smaller than %dKB", kb))
			}
			return next(c)
		}
	}
}

// AddHeadersMiddleware レスポンスヘッダーを追加するミドルウェア
func AddHeadersMiddleware(headers map[string]string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for k, v := range headers {
				c.Response().Header().Set(k, v)
			}
			return next(c)
		}
	}
}

/*
func CheckModTimePreconditionMiddleware(modTimeFunc func(c echo.Context) time.Time, preFunc ...echo.HandlerFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if len(preFunc) > 0 {
				if err := preFunc[0](c); err != nil {
					return err
				}
			}
			modTime := modTimeFunc(c)
			setLastModified(c, modTime)
			if ok, _ := checkPreconditions(c, modTime); ok {
				return nil
			}
			return next(c)
		}
	}
}
*/

// ValidateGroupID 'groupID'パラメータのグループを検証するミドルウェア
func (h *Handlers) ValidateGroupID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			groupID := getRequestParamAsUUID(c, paramGroupID)

			g, err := h.Repo.GetUserGroup(groupID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			c.Set("paramGroup", g)
			return next(c)
		}
	}
}

func getGroupFromContext(c echo.Context) *model.UserGroup {
	return c.Get("paramGroup").(*model.UserGroup)
}

// ValidateStampID 'stampID'パラメータのスタンプを検証するミドルウェア
func (h *Handlers) ValidateStampID(existenceCheckOnly bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			stampID := getRequestParamAsUUID(c, paramStampID)

			if existenceCheckOnly {
				if ok, err := h.Repo.StampExists(stampID); err != nil {
					return internalServerError(err, h.requestContextLogger(c))
				} else if !ok {
					return notFound()
				}
				return next(c)
			}

			s, err := h.Repo.GetStamp(stampID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			c.Set("paramStamp", s)
			return next(c)
		}
	}
}

func getStampFromContext(c echo.Context) *model.Stamp {
	return c.Get("paramStamp").(*model.Stamp)
}

// ValidateMessageID 'messageID'パラメータのメッセージを検証するミドルウェア
func (h *Handlers) ValidateMessageID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			messageID := getRequestParamAsUUID(c, paramMessageID)
			userID := getRequestUserID(c)

			m, err := h.Repo.GetMessageByID(messageID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			} else if !ok {
				return notFound()
			}

			c.Set("paramMessage", m)
			return next(c)
		}
	}
}

func getMessageFromContext(c echo.Context) *model.Message {
	return c.Get("paramMessage").(*model.Message)
}

// ValidatePinID 'pinID'パラメータのピンを検証するミドルウェア
func (h *Handlers) ValidatePinID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			pinID := getRequestParamAsUUID(c, paramPinID)

			pin, err := h.Repo.GetPin(pinID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			if pin.Message.ID == uuid.Nil {
				return notFound()
			}

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, pin.Message.ChannelID); err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			} else if !ok {
				return notFound()
			}

			c.Set("paramPin", pin)
			return next(c)
		}
	}
}

func getPinFromContext(c echo.Context) *model.Pin {
	return c.Get("paramPin").(*model.Pin)
}

// ValidateClipID 'clipID'パラメータのクリップを検証するミドルウェア
func (h *Handlers) ValidateClipID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			clipID := getRequestParamAsUUID(c, paramClipID)

			clip, err := h.Repo.GetClipMessage(clipID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			// クリップがリクエストユーザーのものかを確認
			if clip.UserID != userID {
				return notFound()
			}

			c.Set("paramClip", clip)
			return next(c)
		}
	}
}

func getClipFromContext(c echo.Context) *model.Clip {
	return c.Get("paramClip").(*model.Clip)
}

// ValidateClipFolderID 'folderID'パラメータのクリップフォルダを検証するミドルウェア
func (h *Handlers) ValidateClipFolderID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			folderID := getRequestParamAsUUID(c, paramFolderID)

			folder, err := h.Repo.GetClipFolder(folderID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			// フォルダがリクエストユーザーのものかを確認
			if folder.UserID != userID {
				return notFound()
			}

			c.Set("paramClipFolder", folder)
			return next(c)
		}
	}
}

func getClipFolderFromContext(c echo.Context) *model.ClipFolder {
	return c.Get("paramClipFolder").(*model.ClipFolder)
}

// ValidateChannelID 'channelID'パラメータのチャンネルを検証するミドルウェア
func (h *Handlers) ValidateChannelID(availabilityCheckOnly bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			channelID := getRequestParamAsUUID(c, paramChannelID)

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, channelID); err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			} else if !ok {
				return notFound()
			}

			if availabilityCheckOnly {
				return next(c)
			}

			ch, err := h.Repo.GetChannel(channelID)
			if err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			}

			c.Set("paramChannel", ch)
			return next(c)
		}
	}
}

func getChannelFromContext(c echo.Context) *model.Channel {
	return c.Get("paramChannel").(*model.Channel)
}

// ValidateUserID 'userID'パラメータのユーザーを検証するミドルウェア
func (h *Handlers) ValidateUserID(existenceCheckOnly bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestParamAsUUID(c, paramUserID)

			if existenceCheckOnly {
				if ok, err := h.Repo.UserExists(userID); err != nil {
					return internalServerError(err, h.requestContextLogger(c))
				} else if !ok {
					return notFound()
				}
				return next(c)
			}

			user, err := h.Repo.GetUser(userID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			c.Set("paramUser", user)
			return next(c)
		}
	}
}

func getUserFromContext(c echo.Context) *model.User {
	return c.Get("paramUser").(*model.User)
}

// ValidateWebhookID 'webhookID'パラメータのWebhookを検証するミドルウェア
func (h *Handlers) ValidateWebhookID(requestUserCheck bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			webhookID := getRequestParamAsUUID(c, paramWebhookID)

			w, err := h.Repo.GetWebhook(webhookID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			if requestUserCheck {
				user, ok := c.Get("user").(*model.User)
				if !ok || (!h.RBAC.IsGranted(user.ID, user.Role, permission.AccessOthersWebhook) && w.GetCreatorID() != user.ID) {
					return forbidden()
				}
			}

			c.Set("paramWebhook", w)
			return next(c)
		}
	}
}

func getWebhookFromContext(c echo.Context) model.Webhook {
	return c.Get("paramWebhook").(model.Webhook)
}

// ValidateBotID 'botID'パラメータのBotを検証するミドルウェア
func (h *Handlers) ValidateBotID(requestUserCheck bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			botID := getRequestParamAsUUID(c, paramBotID)

			b, err := h.Repo.GetBotByID(botID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			if requestUserCheck {
				user, ok := c.Get("user").(*model.User)
				if !ok || b.CreatorID != user.ID {
					return forbidden()
				}
			}

			c.Set("paramBot", b)
			return next(c)
		}
	}
}

func getBotFromContext(c echo.Context) *model.Bot {
	return c.Get("paramBot").(*model.Bot)
}

// ValidateFileID 'fileID'パラメータのファイルを検証するミドルウェア
func (h *Handlers) ValidateFileID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			fileID := getRequestParamAsUUID(c, paramFileID)

			// アクセス権確認
			if ok, err := h.Repo.IsFileAccessible(fileID, userID); err != nil {
				switch err {
				case repository.ErrNilID, repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			} else if !ok {
				return forbidden()
			}

			meta, err := h.Repo.GetFileMeta(fileID)
			if err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			}

			c.Set("paramFile", meta)
			return next(c)
		}
	}
}

func getFileFromContext(c echo.Context) *model.File {
	return c.Get("paramFile").(*model.File)
}

// ValidateClientID 'clientID'パラメータのクライアントを検証するミドルウェア
func (h *Handlers) ValidateClientID(requestUserCheck bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientID := c.Param("clientID")

			oc, err := h.Repo.GetClient(clientID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return notFound()
				default:
					return internalServerError(err, h.requestContextLogger(c))
				}
			}

			if requestUserCheck {
				userID := getRequestUserID(c)
				if oc.CreatorID != userID {
					return forbidden()
				}
			}

			c.Set("paramClient", oc)
			return next(c)
		}
	}
}

func getClientFromContext(c echo.Context) *model.OAuth2Client {
	return c.Get("paramClient").(*model.OAuth2Client)
}
