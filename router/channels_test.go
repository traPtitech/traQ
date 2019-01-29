package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGroup_Channels(t *testing.T) {
	assert, require, session, adminSession := beforeTest(t)

	t.Run("TestGetChannels", func(t *testing.T) {
		// パラレルにしない

		for i := 0; i < 5; i++ {
			mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		}
		mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{testUser.GetUID()})

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			arr := e.GET("/api/1.0/channels").
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array()
			arr.Length().Equal(6 + 1)
		})

		t.Run("Successful2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			arr := e.GET("/api/1.0/channels").
				WithCookie(sessions.CookieName, adminSession).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array()
			arr.Length().Equal(5 + 1)
		})
	})

	// ここから並列テスト

	t.Run("TestPostChannels", func(t *testing.T) {
		t.Parallel()

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/channels").
				WithJSON(&PostChannel{Name: "forbidden", Parent: ""}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)

			cname1 := utils.RandAlphabetAndNumberString(20)
			obj := e.POST("/api/1.0/channels").
				WithCookie(sessions.CookieName, session).
				WithJSON(&PostChannel{Name: cname1, Parent: ""}).
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

			c1, err := model.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
			require.NoError(err)

			cname2 := utils.RandAlphabetAndNumberString(20)
			obj = e.POST("/api/1.0/channels").
				WithCookie(sessions.CookieName, session).
				WithJSON(&PostChannel{Name: cname2, Parent: c1.ID.String()}).
				Expect().
				Status(http.StatusCreated).
				JSON().
				Object()

			obj.Value("channelId").String().NotEmpty()
			obj.Value("name").String().Equal(cname2)
			obj.Value("visibility").Boolean().True()
			obj.Value("parent").String().Equal(c1.ID.String())
			obj.Value("force").Boolean().False()
			obj.Value("private").Boolean().False()
			obj.Value("dm").Boolean().False()
			obj.Value("member").Array().Empty()

			_, err = model.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
			require.NoError(err)
		})

		t.Run("Successful2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)

			user2 := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).GetUID()
			cname := utils.RandAlphabetAndNumberString(20)
			obj := e.POST("/api/1.0/channels").
				WithCookie(sessions.CookieName, session).
				WithJSON(&PostChannel{
					Private: true,
					Name:    cname,
					Parent:  "",
					Members: []uuid.UUID{
						testUser.GetUID(),
						user2,
					},
				}).
				Expect().
				Status(http.StatusCreated).
				JSON().
				Object()

			obj.Value("channelId").String().NotEmpty()
			obj.Value("name").String().Equal(cname)
			obj.Value("visibility").Boolean().True()
			obj.Value("parent").String().Empty()
			obj.Value("force").Boolean().False()
			obj.Value("private").Boolean().True()
			obj.Value("dm").Boolean().False()
			obj.Value("member").Array().ContainsOnly(testUser.GetUID(), user2)

			c, err := model.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
			require.NoError(err)

			ok, err := model.IsChannelAccessibleToUser(testUser.GetUID(), c.ID)
			require.NoError(err)
			assert.True(ok)

			ok, err = model.IsChannelAccessibleToUser(user2, c.ID)
			require.NoError(err)
			assert.True(ok)

			ok, err = model.IsChannelAccessibleToUser(uuid.NewV4(), c.ID)
			require.NoError(err)
			assert.False(ok)
		})
	})

	t.Run("PostChannelChildren", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, model.ServerUser().GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/channels/{channelID}/children", pubCh.ID.String()).
				WithJSON(map[string]string{"name": "forbidden"}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)

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

			_, err := model.GetChannel(uuid.FromStringOrNil(obj.Value("channelId").String().Raw()))
			require.NoError(err)
		})
	})

	t.Run("TestGetChannelsByChannelID", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("NotFound", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}", uuid.NewV4().String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNotFound)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			obj := e.GET("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			obj.Value("channelId").String().Equal(pubCh.ID.String())
			obj.Value("name").String().Equal(pubCh.Name)
			obj.Value("visibility").Boolean().Equal(pubCh.IsVisible)
			obj.Value("parent").String().Equal(pubCh.ParentID)
			obj.Value("force").Boolean().Equal(pubCh.IsForced)
			obj.Value("private").Boolean().Equal(!pubCh.IsPublic)
			obj.Value("dm").Boolean().Equal(pubCh.IsDMChannel())
			obj.Value("member").Array().Empty()
		})
	})

	t.Run("TestPatchChannelsByChannelID", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, model.ServerUser().GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithJSON(map[string]interface{}{"name": "renamed", "visibility": true}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)

			newName := utils.RandAlphabetAndNumberString(20)
			e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithCookie(sessions.CookieName, adminSession).
				WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
				Expect().
				Status(http.StatusNoContent)

			ch, err := model.GetChannel(pubCh.ID)
			require.NoError(err)
			assert.Equal(newName, ch.Name)
			assert.False(ch.IsVisible)
			assert.True(ch.IsForced)
		})

		// 権限がない
		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)

			newName := utils.RandAlphabetAndNumberString(20)
			e.PATCH("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]interface{}{"name": newName, "visibility": false, "force": true}).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestDeleteChannelsByChannelID", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithCookie(sessions.CookieName, adminSession).
				Expect().
				Status(http.StatusNoContent)

			_, err := model.GetChannel(pubCh.ID)
			assert.Equal(err, model.ErrNotFound)
		})

		// 権限がない
		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/channels/{channelID}", pubCh.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestPutChannelParent", func(t *testing.T) {
		t.Parallel()

		pCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		cCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
				WithCookie(sessions.CookieName, adminSession).
				WithJSON(map[string]string{"parent": pCh.ID.String()}).
				Expect().
				Status(http.StatusNoContent)

			ch, err := model.GetChannel(cCh.ID)
			require.NoError(err)
			assert.Equal(ch.ParentID, pCh.ID.String())
		})

		// 権限がない
		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/parent", cCh.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"parent": pCh.ID.String()}).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestGetTopic", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		topicText := "Topic test"
		require.NoError(model.UpdateChannelTopic(pubCh.ID, topicText, testUser.GetUID()))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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
	})

	t.Run("TestPutTopic", func(t *testing.T) {
		t.Parallel()

		pubCh := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		topicText := "Topic test"
		require.NoError(model.UpdateChannelTopic(pubCh.ID, topicText, testUser.GetUID()))
		newTopic := "new Topic"

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
				WithJSON(map[string]string{"text": newTopic}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/topic", pubCh.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"text": newTopic}).
				Expect().
				Status(http.StatusNoContent)

			ch, err := model.GetChannel(pubCh.ID)
			require.NoError(err)
			assert.Equal(newTopic, ch.Topic)
		})
	})
}
