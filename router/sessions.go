package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"time"
)

// GetMySessions GET /users/me/sessions
func GetMySessions(c echo.Context) error {
	user := c.Get("user").(*model.User)

	ses, err := sessions.GetByUserID(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
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
func DeleteAllMySessions(c echo.Context) error {
	user := c.Get("user").(*model.User)

	err := sessions.DestroyByUserID(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMySession DELETE /users/me/sessions/:referenceID
func DeleteMySession(c echo.Context) error {
	user := c.Get("user").(*model.User)

	err := sessions.DestroyByReferenceID(user.GetUID(), uuid.FromStringOrNil(c.Param("referenceID")))
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
