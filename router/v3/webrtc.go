package v3

import (
	"encoding/base64"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/service/webrtcv3"
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

// GetWebRTCState GET /webrtc/state
func (h *Handlers) GetWebRTCState(c echo.Context) error {
	type StateSession struct {
		State     string `json:"state"`
		SessionID string `json:"sessionId"`
	}
	type WebRTCUserState struct {
		UserID    uuid.UUID      `json:"userId"`
		ChannelID uuid.UUID      `json:"channelId"`
		Sessions  []StateSession `json:"sessions"`
	}

	res := make([]WebRTCUserState, 0)
	h.WebRTC.IterateStates(func(state webrtcv3.ChannelState) {
		for _, userState := range state.Users() {
			var sessions []StateSession
			for sessionID, state := range userState.Sessions() {
				sessions = append(sessions, StateSession{State: state, SessionID: sessionID})
			}
			res = append(res, WebRTCUserState{
				UserID:    userState.UserID(),
				ChannelID: userState.ChannelID(),
				Sessions:  sessions,
			})
		}
	})

	return c.JSON(http.StatusOK, res)
}
