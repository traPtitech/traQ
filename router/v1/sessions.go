package v1

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"net/http"
	"time"
)

// GetMySessions GET /users/me/sessions
func (h *Handlers) GetMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	ses, err := sessions.GetByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ID            string    `json:"id"`
		LastIP        string    `json:"lastIP"`
		LastUserAgent string    `json:"lastUserAgent"`
		LastAccess    time.Time `json:"lastAccess"`
		CreatedAt     time.Time `json:"createdAt"`
	}

	res := make([]response, len(ses))
	for k, v := range ses {
		referenceID, created, lastAccess, lastIP, lastUserAgent := v.GetSessionInfo()
		res[k] = response{
			ID:            referenceID.String(),
			LastIP:        lastIP,
			LastUserAgent: lastUserAgent,
			LastAccess:    lastAccess,
			CreatedAt:     created,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// DeleteAllMySessions DELETE /users/me/sessions
func (h *Handlers) DeleteAllMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	err := sessions.DestroyByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMySession DELETE /users/me/sessions/:referenceID
func (h *Handlers) DeleteMySession(c echo.Context) error {
	userID := getRequestUserID(c)
	referenceID := getRequestParamAsUUID(c, consts.ParamReferenceID)

	err := sessions.DestroyByReferenceID(userID, referenceID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
