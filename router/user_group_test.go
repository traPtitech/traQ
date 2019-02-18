package router

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"
)

func TestHandlers_GetUserGroups(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, s1)

	mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustMakeUserGroup(t, repo, random, uuid.Nil)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups").
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(3)
	})
}

func TestHandlers_PostUserGroups(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		name := utils.RandAlphabetAndNumberString(20)
		e.POST("/api/1.0/groups").
			WithJSON(map[string]interface{}{"name": name, "description": name}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		name := utils.RandAlphabetAndNumberString(20)
		mustMakeUserGroup(t, repo, name, uuid.Nil)
		e.POST("/api/1.0/groups").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": name, "description": name}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		name := utils.RandAlphabetAndNumberString(20)
		obj := e.POST("/api/1.0/groups").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": name, "description": name}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("groupId").String().NotEmpty()
		obj.Value("name").String().Equal(name)
		obj.Value("description").String().Equal(name)
		obj.Value("adminUserId").String().Equal(user.ID.String())
		obj.Value("members").Array().Empty()
	})
}

func TestHandlers_GetUserGroup(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	g := mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustAddUserToGroup(t, repo, user.ID, g.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups/{groupID}", g.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups/{groupID}", uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("groupId").String().Equal(g.ID.String())
		obj.Value("name").String().Equal(g.Name)
		obj.Value("description").String().Equal(g.Description)
		obj.Value("adminUserId").String().Equal(g.AdminUserID.String())
		obj.Value("members").Array().ContainsOnly(user.ID.String())
	})
}

func TestHandlers_PatchUserGroup(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	user2 := mustMakeUser(t, repo, random)
	g := mustMakeUserGroup(t, repo, random, user.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/groups/{groupID}", uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"description": "aaa"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": true, "adminUserId": uuid.Nil.String()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"adminUserId": uuid.Nil.String()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		name := utils.RandAlphabetAndNumberString(20)
		mustMakeUserGroup(t, repo, name, uuid.Nil)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": name}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, user2.ID)).
			WithJSON(map[string]interface{}{"description": "aaa"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		g := mustMakeUserGroup(t, repo, random, user.ID)
		name := utils.RandAlphabetAndNumberString(20)
		e.PATCH("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"name": name, "description": "aaa", "adminUserId": user2.ID}).
			Expect().
			Status(http.StatusNoContent)

		a, err := repo.GetUserGroup(g.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, a.Name, name)
			assert.Equal(t, a.Description, "aaa")
			assert.Equal(t, a.AdminUserID, user2.ID)
		}
	})

}

func TestHandlers_DeleteUserGroup(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	g := mustMakeUserGroup(t, repo, random, user.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}", g.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}", uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		user2 := mustMakeUser(t, repo, random)
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, user2.ID)).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.ID)
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetUserGroup(g.ID)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}

func TestHandlers_GetUserGroupMembers(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	g := mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustAddUserToGroup(t, repo, user.ID, g.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups/{groupID}/members", g.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups/{groupID}/members", uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(user.ID.String())
	})
}

func TestHandlers_PostUserGroupMembers(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)
	g := mustMakeUserGroup(t, repo, random, user.ID)
	user2 := mustMakeUser(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithJSON(map[string]interface{}{"userId": user.ID}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"userId": user.ID}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"userId": true}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, user2.ID)).
			WithJSON(map[string]interface{}{"userId": user.ID}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"userId": uuid.NewV4()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/groups/{groupID}/members", g.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"userId": user.ID}).
			Expect().
			Status(http.StatusNoContent)

		ids, err := repo.GetUserGroupMemberIDs(g.ID)
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, ids, []uuid.UUID{user.ID})
		}
	})
}

func TestHandlers_DeleteUserGroupMembers(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)
	g := mustMakeUserGroup(t, repo, random, user.ID)
	mustAddUserToGroup(t, repo, user.ID, g.ID)
	user2 := mustMakeUser(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}/members/{userID}", g.ID.String(), user.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}/members/{userID}", uuid.NewV4().String(), user.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}/members/{userID}", g.ID.String(), uuid.NewV4().String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}/members/{userID}", g.ID.String(), user.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, user2.ID)).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/groups/{groupID}/members/{userID}", g.ID.String(), user.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		ids, err := repo.GetUserGroupMemberIDs(g.ID)
		if assert.NoError(t, err) {
			assert.Len(t, ids, 0)
		}
	})
}

func TestHandlers_GetMyBelongingGroup(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common5)

	g1 := mustMakeUserGroup(t, repo, random, uuid.Nil)
	g2 := mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustAddUserToGroup(t, repo, user.ID, g1.ID)
	mustAddUserToGroup(t, repo, user.ID, g2.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/groups").
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/groups").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(g1.ID.String(), g2.ID.String())
	})
}

func TestHandlers_GetUserBelongingGroup(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common5)

	user := mustMakeUser(t, repo, random)
	g1 := mustMakeUserGroup(t, repo, random, uuid.Nil)
	g2 := mustMakeUserGroup(t, repo, random, uuid.Nil)
	mustAddUserToGroup(t, repo, user.ID, g1.ID)
	mustAddUserToGroup(t, repo, user.ID, g2.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/groups", user.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/groups", uuid.NewV4()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/groups", user.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(g1.ID.String(), g2.ID.String())
	})
}
