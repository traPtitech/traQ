package v3

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/router/session"
)

func TestPostWebRTCAuthenticateRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		PeerID string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty peer id",
			fields{PeerID: ""},
			true,
		},
		{
			"success",
			fields{PeerID: uuid.Must(uuid.NewV4()).String()},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostWebRTCAuthenticateRequest{
				PeerID: tt.fields.PeerID,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_PostWebRTCAuthenticate(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webrtc/authenticate"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostWebRTCAuthenticateRequest{PeerID: user.GetID().String()}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostWebRTCAuthenticateRequest{PeerID: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostWebRTCAuthenticateRequest{PeerID: user.GetID().String()}).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("peerId").String().Equal(user.GetID().String())
		obj.Value("ttl").Number()
		obj.Value("timestamp").Number()
		obj.Value("authToken").String().NotEmpty()
	})
}
