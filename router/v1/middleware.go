package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"golang.org/x/sync/singleflight"
)

// ValidateGroupID 'groupID'パラメータのグループを検証するミドルウェア
func (h *Handlers) ValidateGroupID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			groupID := getRequestParamAsUUID(c, consts.ParamGroupID)

			g, err := h.Repo.GetUserGroup(groupID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
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
			stampID := getRequestParamAsUUID(c, consts.ParamStampID)

			if existenceCheckOnly {
				if ok, err := h.Repo.StampExists(stampID); err != nil {
					return herror.InternalServerError(err)
				} else if !ok {
					return herror.NotFound()
				}
				return next(c)
			}

			s, err := h.Repo.GetStamp(stampID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
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
	var cache singleflight.Group

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			messageID := getRequestParamAsUUID(c, consts.ParamMessageID)
			userID := getRequestUserID(c)

			mI, err, _ := cache.Do(messageID.String(), func() (interface{}, error) { return h.Repo.GetMessageByID(messageID) })
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			}

			m := mI.(*model.Message)
			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
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
			pinID := getRequestParamAsUUID(c, consts.ParamPinID)

			pin, err := h.Repo.GetPin(pinID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			}

			if pin.Message.ID == uuid.Nil {
				return herror.NotFound()
			}

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, pin.Message.ChannelID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			c.Set("paramPin", pin)
			return next(c)
		}
	}
}

func getPinFromContext(c echo.Context) *model.Pin {
	return c.Get("paramPin").(*model.Pin)
}

// ValidateChannelID 'channelID'パラメータのチャンネルを検証するミドルウェア
func (h *Handlers) ValidateChannelID(availabilityCheckOnly bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := getRequestUserID(c)
			channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

			if ok, err := h.Repo.IsChannelAccessibleToUser(userID, channelID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			if availabilityCheckOnly {
				return next(c)
			}

			ch, err := h.Repo.GetChannel(channelID)
			if err != nil {
				return herror.InternalServerError(err)
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
			userID := getRequestParamAsUUID(c, consts.ParamUserID)

			if existenceCheckOnly {
				if ok, err := h.Repo.UserExists(userID); err != nil {
					return herror.InternalServerError(err)
				} else if !ok {
					return herror.NotFound()
				}
				return next(c)
			}

			user, err := h.Repo.GetUser(userID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
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
			webhookID := getRequestParamAsUUID(c, consts.ParamWebhookID)

			w, err := h.Repo.GetWebhook(webhookID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			}

			if requestUserCheck {
				user, ok := c.Get("user").(*model.User)
				if !ok || (!h.RBAC.IsGranted(user.Role, permission.AccessOthersWebhook) && w.GetCreatorID() != user.ID) {
					return herror.Forbidden()
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
			botID := getRequestParamAsUUID(c, consts.ParamBotID)

			b, err := h.Repo.GetBotByID(botID)
			if err != nil {
				switch err {
				case repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			}

			if requestUserCheck {
				user, ok := c.Get("user").(*model.User)
				if !ok || (!h.RBAC.IsGranted(user.Role, permission.AccessOthersBot) && b.CreatorID != user.ID) {
					return herror.Forbidden()
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
			fileID := getRequestParamAsUUID(c, consts.ParamFileID)

			// アクセス権確認
			if ok, err := h.Repo.IsFileAccessible(fileID, userID); err != nil {
				switch err {
				case repository.ErrNilID, repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			} else if !ok {
				return herror.Forbidden()
			}

			meta, err := h.Repo.GetFileMeta(fileID)
			if err != nil {
				return herror.InternalServerError(err)
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
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			}

			if requestUserCheck {
				userID := getRequestUserID(c)
				if oc.CreatorID != userID {
					return herror.Forbidden()
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
