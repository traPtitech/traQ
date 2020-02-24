package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
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

			c.Set(consts.KeyParamGroup, g)
			return next(c)
		}
	}
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
