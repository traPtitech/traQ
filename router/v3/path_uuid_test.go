package v3

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_PathUUIDValidation(t *testing.T) {
	t.Parallel()
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	group := env.CreateUserGroup(t, rand, "", "", user.GetID())
	folder := env.CreateClipFolder(t, rand, "", user.GetID())
	token := env.S(t, user.GetID())
	for _, tc := range []struct {
		name, method, path string
		body               map[string]any
		missingStatus      int
	}{
		{"star", http.MethodDelete, "/api/v3/users/me/stars/{id}", nil, http.StatusNoContent},
		{"unread", http.MethodDelete, "/api/v3/users/me/unread/{id}", nil, http.StatusNoContent},
		{"member update", http.MethodPatch, "/api/v3/groups/" + group.ID.String() + "/members/{id}", map[string]any{"role": "member"}, http.StatusBadRequest},
		{"member removal", http.MethodDelete, "/api/v3/groups/" + group.ID.String() + "/members/{id}", nil, http.StatusNoContent},
		{"admin removal", http.MethodDelete, "/api/v3/groups/" + group.ID.String() + "/admins/{id}", nil, http.StatusNoContent},
		{"clip", http.MethodDelete, "/api/v3/clip-folders/" + folder.ID.String() + "/messages/{id}", nil, http.StatusNoContent},
		{"user tag update", http.MethodPatch, "/api/v3/users/" + user.GetID().String() + "/tags/{id}", map[string]any{"isLocked": true}, http.StatusNotFound},
		{"my tag update", http.MethodPatch, "/api/v3/users/me/tags/{id}", map[string]any{"isLocked": true}, http.StatusNotFound},
		{"user tag removal", http.MethodDelete, "/api/v3/users/" + user.GetID().String() + "/tags/{id}", nil, http.StatusNoContent},
		{"my tag removal", http.MethodDelete, "/api/v3/users/me/tags/{id}", nil, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range []struct {
				name, value string
				status      int
			}{
				{"malformed", "invalid", http.StatusNotFound},
				{"nil", uuid.Nil.String(), http.StatusNotFound},
				{"missing v4", uuid.Must(uuid.NewV4()).String(), tc.missingStatus},
				{"missing v7", uuid.Must(uuid.NewV7()).String(), tc.missingStatus},
			} {
				t.Run(id.name, func(t *testing.T) {
					req := env.R(t).Request(tc.method, tc.path, id.value).WithCookie(session.CookieName, token)
					if tc.body != nil {
						req.WithJSON(tc.body)
					}
					req.Expect().Status(id.status)
				})
			}
		})
	}
}
