package router

import (
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"

	"github.com/traPtitech/traQ/oauth2"

	"fmt"

	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

// UserAuthenticate User認証するミドルウェア
func (h *Handlers) UserAuthenticate(oh *oauth2.Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var user *model.User
			ah := c.Request().Header.Get(echo.HeaderAuthorization)
			if len(ah) > 0 {
				// AuthorizationヘッダーがあるためOAuth2で検証

				// Authorizationスキーム検証
				l := len(oauth2.AuthScheme)
				if !(len(ah) > l+1 && ah[:l] == oauth2.AuthScheme) {
					return echo.NewHTTPError(http.StatusUnauthorized, "the Authorization Header's scheme is invalid")
				}

				// OAuth2 Token検証
				token, err := oh.GetTokenByAccess(ah[l+1:])
				if err != nil {
					switch err {
					case oauth2.ErrTokenNotFound:
						return echo.NewHTTPError(http.StatusUnauthorized, "the token is invalid")
					default:
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
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
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}

				// 認可に基づきRole生成
				c.Set("role", token.Scopes.GenerateRole())
			} else {
				// Authorizationヘッダーがないためセッションを確認する
				sess, err := sessions.Get(c.Response(), c.Request(), false)
				if err != nil {
					c.Logger().Errorf("Failed to get a session: %v", err)
					return echo.NewHTTPError(http.StatusInternalServerError)
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
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}
			}

			// ユーザーアカウント状態を確認
			switch user.Status {
			case model.UserAccountStatusSuspended:
				return echo.NewHTTPError(http.StatusForbidden, "this account is currently suspended")
			case model.UserAccountStatusValid:
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
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}
				}

				// ユーザー権限検証
				user := c.Get("user").(*model.User)
				for _, v := range p {
					if !rbac.IsGranted(user.ID, user.Role, v) {
						// NG
						return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
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

// ValidateGroupID 'groupID'パラメータのグループを検証するミドルウェア
func (h *Handlers) ValidateGroupID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			groupID := getRequestParamAsUUID(c, paramGroupID)

			g, err := h.Repo.GetUserGroup(groupID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
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
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				} else if !ok {
					return c.NoContent(http.StatusNotFound)
				}
				return next(c)
			}

			s, err := h.Repo.GetStamp(stampID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
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
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			}

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			} else if !ok {
				return c.NoContent(http.StatusNotFound)
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
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			}

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, pin.Message.ChannelID); err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			} else if !ok {
				return c.NoContent(http.StatusNotFound)
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
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			}

			// クリップがリクエストユーザーのものかを確認
			if clip.UserID != userID {
				return c.NoContent(http.StatusNotFound)
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
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			}

			// フォルダがリクエストユーザーのものかを確認
			if folder.UserID != userID {
				return c.NoContent(http.StatusNotFound)
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
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			} else if !ok {
				return c.NoContent(http.StatusNotFound)
			}

			if availabilityCheckOnly {
				return next(c)
			}

			ch, err := h.Repo.GetChannel(channelID)
			if err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
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
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				} else if !ok {
					return c.NoContent(http.StatusNotFound)
				}
				return next(c)
			}

			user, err := h.Repo.GetUser(userID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
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

			if webhookID == uuid.Nil {
				return c.NoContent(http.StatusNotFound)
			}

			w, err := h.Repo.GetWebhook(webhookID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			}

			if requestUserCheck {
				user, ok := c.Get("user").(*model.User)
				if !ok || w.GetCreatorID() != user.ID {
					return c.NoContent(http.StatusForbidden)
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
					return c.NoContent(http.StatusNotFound)
				default:
					c.Logger().Error(err)
					return c.NoContent(http.StatusInternalServerError)
				}
			} else if !ok {
				return c.NoContent(http.StatusForbidden)
			}

			return next(c)
		}
	}
}
