package v1

import (
	"encoding/base64"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/realtime/webrtc"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/hmac"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/set"
	"net/http"
	"time"
)

// PostSkyWayAuthenticateRequest POST /skyway/authenticate リクエストボディ
type PostSkyWayAuthenticateRequest struct {
	PeerID string `json:"peerId"`
}

func (r PostSkyWayAuthenticateRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.PeerID, vd.Required),
	)
}

// PostSkyWayAuthenticate POST /skyway/authenticate
func (h *Handlers) PostSkyWayAuthenticate(c echo.Context) error {
	var req PostSkyWayAuthenticateRequest
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

// GetChannelWebRTCState GET /channels/:channelID/webrtc/state
func (h *Handlers) GetChannelWebRTCState(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)
	cs := h.Realtime.WebRTC.GetChannelState(channelID)

	var users []*webrtc.UserState
	for _, v := range cs.Users {
		users = append(users, v)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"users": users,
	})
}

// PutWebRTCStateRequest PUT /webrtc/state リクエストボディ
type PutWebRTCStateRequest struct {
	ChannelID optional.UUID `json:"channelId"`
	State     []string      `json:"state"`
}

// PutChannelWebRTCState PUT /webrtc/state
func (h *Handlers) PutWebRTCState(c echo.Context) error {
	userID := getRequestUserID(c)
	var req PutWebRTCStateRequest
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
