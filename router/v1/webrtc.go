package v1

import (
	"encoding/base64"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/webrtc"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/set"
	"net/http"
	"time"
)

// PostSkyWayAuthenticate POST /skyway/authenticate
func (h *Handlers) PostSkyWayAuthenticate(c echo.Context) error {
	var req struct {
		PeerID string `json:"peerId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if len(req.PeerID) == 0 {
		return herror.BadRequest("empty peerId")
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

// GetChannelWebRTCState GET /channels/:channelID/webrtc/state
func (h *Handlers) GetChannelWebRTCState(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)
	cs := h.Realtime.WebRTC.GetChannelState(channelID)

	var users []*webrtc.UserState
	for _, v := range cs.Users {
		users = append(users, v)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// PutChannelWebRTCState PUT /webrtc/state
func (h *Handlers) PutWebRTCState(c echo.Context) error {
	userID := getRequestUserID(c)
	var req struct {
		ChannelID uuid.NullUUID `json:"channelId"`
		State     []string      `json:"state"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Realtime.WebRTC.SetState(userID, req.ChannelID.UUID, set.StringSetFromArray(req.State)); err != nil {
		return herror.BadRequest(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetWebRTCState GET /webrtc/state
func (h *Handlers) GetWebRTCState(c echo.Context) error {
	userID := getRequestUserID(c)
	us := h.Realtime.WebRTC.GetUserState(userID)
	return c.JSON(http.StatusOK, us)
}
