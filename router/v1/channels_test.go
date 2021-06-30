package v1

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/random"
)

func TestHandlers_PostChannels(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common1)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels").
			WithJSON(&PostChannelRequest{Name: "forbidden", Parent: uuid.Nil}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels").
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)

		cname1 := random.AlphaNumeric(20)
		obj := e.POST("/api/1.0/channels").
			WithCookie(session.CookieName, s).
			WithJSON(&PostChannelRequest{Name: cname1, Parent: uuid.Nil}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("channelId").String().NotEmpty()
		obj.Value("name").String().Equal(cname1)
		obj.Value("visibility").Boolean().True()
		obj.Value("parent").String().Empty()
		obj.Value("force").Boolean().False()
		obj.Value("private").Boolean().False()
		obj.Value("dm").Boolean().False()
		obj.Value("member").Array().Empty()

		c1, err := uuid.FromString(obj.Value("channelId").String().Raw())
		require.NoError(t, err)

		cname2 := random.AlphaNumeric(20)
		obj = e.POST("/api/1.0/channels").
			WithCookie(session.CookieName, s).
			WithJSON(&PostChannelRequest{Name: cname2, Parent: c1}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("channelId").String().NotEmpty()
		obj.Value("name").String().Equal(cname2)
		obj.Value("visibility").Boolean().True()
		obj.Value("parent").String().Equal(c1.String())
		obj.Value("force").Boolean().False()
		obj.Value("private").Boolean().False()
		obj.Value("dm").Boolean().False()
		obj.Value("member").Array().Empty()

		_, err = env.ChannelManager.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
		require.NoError(t, err)
	})
}

func TestHandlers_PostChannelChildren(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common1)

	pubCh := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithJSON(map[string]string{"name": "forbidden"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{"name": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)

		cname1 := random.AlphaNumeric(20)
		obj := e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"name": cname1}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("channelId").String().NotEmpty()
		obj.Value("name").String().Equal(cname1)
		obj.Value("visibility").Boolean().True()
		obj.Value("parent").String().Equal(pubCh.ID.String())
		obj.Value("force").Boolean().False()
		obj.Value("private").Boolean().False()
		obj.Value("dm").Boolean().False()
		obj.Value("member").Array().Empty()

		_, err := env.ChannelManager.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
		require.NoError(t, err)
	})
}

func TestHandlers_GetChannelByChannelID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common1)

	pubCh := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}", uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		obj := e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("channelId").String().Equal(pubCh.ID.String())
		obj.Value("name").String().Equal(pubCh.Name)
		obj.Value("visibility").Boolean().Equal(pubCh.IsVisible)
		obj.Value("parent").String().Equal("")
		obj.Value("force").Boolean().Equal(pubCh.IsForced)
		obj.Value("private").Boolean().Equal(!pubCh.IsPublic)
		obj.Value("dm").Boolean().Equal(pubCh.IsDMChannel())
		obj.Value("member").Array().Empty()
	})
}
