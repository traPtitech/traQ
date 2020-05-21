package v1

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"net/http"
	"testing"
)

func TestHandlers_GetChannels(t *testing.T) {
	t.Parallel()
	env, _, require, s, adminSession := setup(t, s1)

	for i := 0; i < 5; i++ {
		c := env.mustMakeChannel(t, rand)
		_, err := env.Repository.CreatePublicChannel(random.AlphaNumeric(20), c.ID, uuid.Nil)
		require.NoError(err)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		arr := e.GET("/api/1.0/channels").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()
		arr.Length().Equal(10 + 1)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		arr := e.GET("/api/1.0/channels").
			WithCookie(session.CookieName, adminSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()
		arr.Length().Equal(10 + 1)
	})
}

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

		_, err = env.Repository.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
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

		_, err := env.Repository.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
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

func TestHandlers_PatchChannelByChannelID(t *testing.T) {
	t.Parallel()
	env, _, _, s, adminSession := setup(t, common1)

	pubCh := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithJSON(map[string]interface{}{"name": "renamed", "visibility": true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(map[string]interface{}{"name": true, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		assert, require := assertAndRequire(t)

		newName := random.AlphaNumeric(20)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.Repository.GetChannel(pubCh.ID)
		require.NoError(err)
		assert.Equal(newName, ch.Name)
		assert.False(ch.IsVisible)
		assert.True(ch.IsForced)
	})

	// 権限がない
	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)

		newName := random.AlphaNumeric(20)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_PutChannelParent(t *testing.T) {
	t.Parallel()
	env, _, _, s, adminSession := setup(t, common1)

	pCh := env.mustMakeChannel(t, rand)
	cCh := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(map[string]string{"parent": pCh.ID.String()}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.Repository.GetChannel(cCh.ID)
		require.NoError(t, err)
		assert.Equal(t, ch.ParentID, pCh.ID)
	})

	// 権限がない
	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"parent": pCh.ID.String()}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_GetTopic(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common1)

	pubCh := env.mustMakeChannel(t, rand)
	topicText := "Topic test"
	require.NoError(t, env.Repository.UpdateChannel(pubCh.ID, repository.UpdateChannelArgs{
		UpdaterID: testUser.GetID(),
		Topic:     optional.StringFrom(topicText),
	}))

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("text").
			String().
			Equal(topicText)
	})
}

func TestHandlers_PutTopic(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common1)

	pubCh := env.mustMakeChannel(t, rand)
	topicText := "Topic test"
	require.NoError(t, env.Repository.UpdateChannel(pubCh.ID, repository.UpdateChannelArgs{
		UpdaterID: testUser.GetID(),
		Topic:     optional.StringFrom(topicText),
	}))
	newTopic := "new Topic"

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithJSON(map[string]string{"text": newTopic}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{"text": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"text": newTopic}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.Repository.GetChannel(pubCh.ID)
		require.NoError(t, err)
		assert.Equal(t, newTopic, ch.Topic)
	})
}
