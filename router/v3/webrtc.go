package v3

import (
	"encoding/base64"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/utils/hmac"
	"net/http"
	"time"
)

// PostWebRTCAuthenticateRequest POST /webrtc/authenticate リクエストボディ
type PostWebRTCAuthenticateRequest struct {
	PeerID string `json:"peerId"`
}

func (r PostWebRTCAuthenticateRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.PeerID, vd.Required),
	)
}

// PostWebRTCAuthenticate POST /webrtc/authenticate
func (h *Handlers) PostWebRTCAuthenticate(c echo.Context) error {
	if len(h.SkyWaySecretKey) == 0 {
		return echo.NewHTTPError(http.StatusServiceUnavailable)
	}

	var req PostWebRTCAuthenticateRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ts := time.Now().Unix()
	ttl := 40000
	hash := hmac.SHA256([]byte(fmt.Sprintf("%d:%d:%s", ts, ttl, req.PeerID)), h.SkyWaySecretKey)
	return c.JSON(http.StatusOK, echo.Map{
		"peerId":    req.PeerID,
		"timestamp": ts,
		"ttl":       ttl,
		"authToken": base64.StdEncoding.EncodeToString(hash),
	})
}
