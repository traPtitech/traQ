package v1

import (
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// GetMySessions GET /users/me/sessions
func (h *Handlers) GetMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	ses, err := h.SessStore.GetSessionsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
	}

	res := make([]response, len(ses))
	for k, v := range ses {
		res[k] = response{
			ID:        v.RefID(),
			CreatedAt: v.CreatedAt(),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// DeleteAllMySessions DELETE /users/me/sessions
func (h *Handlers) DeleteAllMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	if err := h.SessStore.RevokeSessionsByUserID(userID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMySession DELETE /users/me/sessions/:referenceID
func (h *Handlers) DeleteMySession(c echo.Context) error {
	refID := getRequestParamAsUUID(c, consts.ParamReferenceID)

	if err := h.SessStore.RevokeSessionByRefID(refID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
