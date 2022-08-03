package rbac_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/testUtils"
)

type Repo struct {
	testUtils.EmptyTestRepository
}

func (r *Repo) GetAllUserRoles() ([]*model.UserRole, error) {
	r1 := &model.UserRole{Name: "r1", Permissions: []model.RolePermission{{Permission: "p1"}}}
	r2 := &model.UserRole{Name: "r2", Permissions: []model.RolePermission{{Permission: "p2"}}}
	r3 := &model.UserRole{Name: "r3", Permissions: []model.RolePermission{{Permission: "p3"}}}
	r4 := &model.UserRole{Name: "r4", Permissions: []model.RolePermission{{Permission: "p4"}}}

	r2.Inheritances = append(r2.Inheritances, r3)
	r3.Inheritances = append(r3.Inheritances, r4)

	return []*model.UserRole{r1, r2, r3, r4}, nil
}

func setup(t *testing.T) rbac.RBAC {
	t.Helper()
	repo := new(Repo)
	r, err := rbac.New(repo)
	require.NoError(t, err)
	return r
}

func Test_rbacImpl_IsGranted(t *testing.T) {
	t.Parallel()

	r := setup(t)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		assert.False(t, r.IsGranted("r5", "p1"))
		assert.False(t, r.IsGranted("r2", "p0"))
	})

	t.Run("not granted", func(t *testing.T) {
		t.Parallel()

		assert.False(t, r.IsGranted("r1", "p2"))
		assert.False(t, r.IsGranted("r2", "p1"))
	})

	t.Run("granted", func(t *testing.T) {
		t.Parallel()

		assert.True(t, r.IsGranted("r1", "p1"))
		assert.True(t, r.IsGranted("r2", "p2"))
		assert.True(t, r.IsGranted("r3", "p3"))
		assert.True(t, r.IsGranted("r4", "p4"))
	})

	t.Run("granted via inheritance", func(t *testing.T) {
		t.Parallel()

		assert.True(t, r.IsGranted("r2", "p3"))
		assert.True(t, r.IsGranted("r2", "p4"))
		assert.True(t, r.IsGranted("r3", "p4"))
	})

	t.Run("granted (admin)", func(t *testing.T) {
		t.Parallel()

		assert.True(t, r.IsGranted(role.Admin, "p1"))
	})
}

func Test_rbacImpl_IsAllGranted(t *testing.T) {
	t.Parallel()

	r := setup(t)

	t.Run("not granted", func(t *testing.T) {
		t.Parallel()

		assert.False(t, r.IsAllGranted([]string{"r1"}, "p4"))
		assert.False(t, r.IsAllGranted([]string{"r1", "r2"}, "p1"))
		assert.False(t, r.IsAllGranted([]string{"r2", "r3", "r4"}, "p1"))
	})

	t.Run("granted", func(t *testing.T) {
		t.Parallel()

		assert.True(t, r.IsAllGranted([]string{"r1"}, "p1"))
		assert.True(t, r.IsAllGranted([]string{"r2"}, "p2"))
		assert.True(t, r.IsAllGranted([]string{"r2", "r3"}, "p3"))
		assert.True(t, r.IsAllGranted([]string{"r2", "r3", "r4"}, "p4"))
	})
}

func Test_rbacImpl_IsAnyGranted(t *testing.T) {
	t.Parallel()

	r := setup(t)

	t.Run("not granted", func(t *testing.T) {
		t.Parallel()

		assert.False(t, r.IsAnyGranted([]string{"r1"}, "p2"))
		assert.False(t, r.IsAnyGranted([]string{"r2", "r3", "r4"}, "p1"))
	})

	t.Run("granted", func(t *testing.T) {
		t.Parallel()

		assert.True(t, r.IsAnyGranted([]string{"r1"}, "p1"))
		assert.True(t, r.IsAnyGranted([]string{"r2"}, "p2"))
		assert.True(t, r.IsAnyGranted([]string{"r1", "r2"}, "p1"))
		assert.True(t, r.IsAnyGranted([]string{"r1", "r2"}, "p2"))
	})
}

func Test_rbacImpl_GetGrantedPermissions(t *testing.T) {
	t.Parallel()

	r := setup(t)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		assert.ElementsMatch(t, r.GetGrantedPermissions("r0"), []permission.Permission{})
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.ElementsMatch(t, r.GetGrantedPermissions("r1"), []permission.Permission{"p1"})
		assert.ElementsMatch(t, r.GetGrantedPermissions("r2"), []permission.Permission{"p2", "p3", "p4"})
		assert.ElementsMatch(t, r.GetGrantedPermissions("r3"), []permission.Permission{"p3", "p4"})
		assert.ElementsMatch(t, r.GetGrantedPermissions("r4"), []permission.Permission{"p4"})
	})
}
