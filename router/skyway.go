package router

import (
	"encoding/base64"
	"fmt"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"time"
)

// PostSkyWayAuthenticate POST /skyway/authenticate
func (h *Handlers) PostSkyWayAuthenticate(c echo.Context) error {
	var req struct {
		PeerID string `json:"peerId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if len(req.PeerID) == 0 {
		return badRequest("empty peerId")
	}

	ts := time.Now().Unix()
	ttl := 40000
	hash := utils.CalcHMACSHA256([]byte(fmt.Sprintf("%d:%d:%s", ts, ttl, req.PeerID)), h.SkyWaySecretKey)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"peerId":    req.PeerID,
		"timestamp": ts,
		"ttl":       ttl,
		"authToken": base64.StdEncoding.EncodeToString(hash),
	})
}
