package v3

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/qall"
)

type soundboardCreatorRecorder struct {
	qall.Soundboard
	creatorID uuid.UUID
}

func (s *soundboardCreatorRecorder) SaveSoundboardItem(_ context.Context, _ uuid.UUID, _, _ string, _ model.FileType, _ io.Reader, _ *uuid.UUID, creatorID uuid.UUID) error {
	s.creatorID = creatorID
	return nil
}

func TestHandlers_CreateSoundboardItemCreator(t *testing.T) {
	t.Parallel()
	userID := uuid.Must(uuid.NewV7())
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	require.NoError(t, w.WriteField("creatorId", uuid.Must(uuid.NewV7()).String()))
	file, err := w.CreateFormFile("file", "sound.wav")
	require.NoError(t, err)
	_, err = file.Write([]byte("sound"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
	res := httptest.NewRecorder()
	c := echo.New().NewContext(req, res)
	c.Set(consts.KeyUser, &model.User{ID: userID})
	soundboard := &soundboardCreatorRecorder{}
	h := &Handlers{Soundboard: soundboard}
	require.NoError(t, h.CreateSoundboardItem(c))
	require.Equal(t, http.StatusNoContent, res.Code)
	require.Equal(t, userID, soundboard.creatorID)
}

func TestHandlers_GetQallEndpoints(t *testing.T) {
	t.Parallel()

	path := "/api/v3/qall/endpoints"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	commonSession := env.S(t, user.GetID())

	issueToken := func(t *testing.T, scopes ...model.AccessScope) string {
		t.Helper()
		s := model.AccessScopes{}
		s.Add(scopes...)
		tok, err := env.Repository.IssueToken(context.TODO(), client, user.GetID(), client.RedirectURI, s, 86400, false)
		require.NoError(t, err)
		return tok.AccessToken
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (session)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("forbidden (oauth2 token without qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "write")).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success (oauth2 token with qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "qall")).
			Expect().
			Status(http.StatusOK)
	})
}
