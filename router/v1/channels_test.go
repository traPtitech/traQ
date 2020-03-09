package v1

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"testing"
)

func TestHandlers_GetChannels(t *testing.T) {
	t.Parallel()
	repo, server, _, require, session, adminSession := setup(t, s1)

	for i := 0; i < 5; i++ {
		c := mustMakeChannel(t, repo, random)
		_, err := repo.CreatePublicChannel(utils.RandAlphabetAndNumberString(20), c.ID, uuid.Nil)
		require.NoError(err)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		arr := e.GET("/api/1.0/channels").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()
		arr.Length().Equal(10 + 1)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		arr := e.GET("/api/1.0/channels").
			WithCookie(sessions.CookieName, adminSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()
		arr.Length().Equal(10 + 1)
	})
}

func TestHandlers_PostChannels(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels").
			WithJSON(&PostChannelRequest{Name: "forbidden", Parent: uuid.Nil}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)

		cname1 := utils.RandAlphabetAndNumberString(20)
		obj := e.POST("/api/1.0/channels").
			WithCookie(sessions.CookieName, session).
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

		cname2 := utils.RandAlphabetAndNumberString(20)
		obj = e.POST("/api/1.0/channels").
			WithCookie(sessions.CookieName, session).
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

		_, err = repo.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
		require.NoError(t, err)
	})
}

func TestHandlers_PostChannelChildren(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	pubCh := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithJSON(map[string]string{"name": "forbidden"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)

		cname1 := utils.RandAlphabetAndNumberString(20)
		obj := e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
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

		_, err := repo.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
		require.NoError(t, err)
	})
}

func TestHandlers_GetChannelByChannelID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	pubCh := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
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
	repo, server, _, _, session, adminSession := setup(t, common1)

	pubCh := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithJSON(map[string]interface{}{"name": "renamed", "visibility": true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, adminSession).
			WithJSON(map[string]interface{}{"name": true, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		assert, require := assertAndRequire(t)

		newName := utils.RandAlphabetAndNumberString(20)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, adminSession).
			WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := repo.GetChannel(pubCh.ID)
		require.NoError(err)
		assert.Equal(newName, ch.Name)
		assert.False(ch.IsVisible)
		assert.True(ch.IsForced)
	})

	// 権限がない
	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)

		newName := utils.RandAlphabetAndNumberString(20)
		e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteChannelByChannelID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, adminSession := setup(t, common1)

	pubCh := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, adminSession).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetChannel(pubCh.ID)
		assert.Equal(t, err, repository.ErrNotFound)
	})

	// 権限がない
	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		pubCh := mustMakeChannel(t, repo, random)
		e := makeExp(t, server)
		e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_PutChannelParent(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, adminSession := setup(t, common1)

	pCh := mustMakeChannel(t, repo, random)
	cCh := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			WithCookie(sessions.CookieName, adminSession).
			WithJSON(map[string]string{"parent": pCh.ID.String()}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := repo.GetChannel(cCh.ID)
		require.NoError(t, err)
		assert.Equal(t, ch.ParentID, pCh.ID)
	})

	// 権限がない
	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"parent": pCh.ID.String()}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_GetTopic(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common1)

	pubCh := mustMakeChannel(t, repo, random)
	topicText := "Topic test"
	require.NoError(t, repo.UpdateChannel(pubCh.ID, repository.UpdateChannelArgs{
		UpdaterID: testUser.ID,
		Topic:     null.StringFrom(topicText),
	}))

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
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
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common1)

	pubCh := mustMakeChannel(t, repo, random)
	topicText := "Topic test"
	require.NoError(t, repo.UpdateChannel(pubCh.ID, repository.UpdateChannelArgs{
		UpdaterID: testUser.ID,
		Topic:     null.StringFrom(topicText),
	}))
	newTopic := "new Topic"

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithJSON(map[string]string{"text": newTopic}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"text": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"text": newTopic}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := repo.GetChannel(pubCh.ID)
		require.NoError(t, err)
		assert.Equal(t, newTopic, ch.Topic)
	})
}
