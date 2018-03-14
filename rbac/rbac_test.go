package rbac

import (
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	rbac := New(nil)
	if assert.NotNil(rbac) {
		assert.NotNil(rbac.overrides)
	}
}

func TestRBAC_IsGranted(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	rbac := New(nil)
	u1 := uuid.NewV4()
	rA := gorbac.NewStdRole("role-a")
	rB := gorbac.NewStdRole("role-b")
	pA := gorbac.NewStdPermission("permission-a")
	pB := gorbac.NewStdPermission("permission-b")
	pC := gorbac.NewStdPermission("permission-c")

	require.NoError(rA.Assign(pA))
	require.NoError(rB.Assign(pA))
	require.NoError(rB.Assign(pB))
	require.NoError(rB.Assign(pC))
	require.NoError(rbac.Add(rA))
	require.NoError(rbac.Add(rB))

	rbac.SetOverride(u1, pB, true)
	rbac.SetOverride(u1, pC, false)

	assert.True(rbac.IsGranted(uuid.Nil, "role-a", pA))
	assert.True(rbac.IsGranted(u1, "role-a", pB))
	assert.False(rbac.IsGranted(uuid.Nil, "role-a", pB))
	assert.False(rbac.IsGranted(uuid.Nil, "role-a", pC))
	assert.True(rbac.IsGranted(uuid.Nil, "role-b", pA))
	assert.True(rbac.IsGranted(uuid.Nil, "role-b", pB))
	assert.True(rbac.IsGranted(uuid.Nil, "role-b", pC))
	assert.False(rbac.IsGranted(u1, "role-b", pC))
}

func TestRBAC_Override(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	rbac := New(nil)
	u1 := uuid.NewV4()
	rA := gorbac.NewStdRole("role-a")
	rB := gorbac.NewStdRole("role-b")
	pA := gorbac.NewStdPermission("permission-a")
	pB := gorbac.NewStdPermission("permission-b")
	pC := gorbac.NewStdPermission("permission-c")

	require.NoError(rA.Assign(pA))
	require.NoError(rB.Assign(pA))
	require.NoError(rB.Assign(pB))
	require.NoError(rB.Assign(pC))
	require.NoError(rbac.Add(rA))
	require.NoError(rbac.Add(rB))

	rbac.SetOverride(u1, pB, true)
	rbac.SetOverride(u1, pC, false)

	assert.Len(rbac.GetOverride(uuid.Nil), 0)
	if or := rbac.GetOverride(u1); assert.Len(or, 2) {
		assert.True(or[pB])
		assert.False(or[pC])
	}

	rbac.DeleteOverride(u1, pC)
	if or := rbac.GetOverride(u1); assert.Len(or, 1) {
		assert.True(or[pB])
	}

	rbac.DeleteOverride(u1, pB)
	assert.Len(rbac.GetOverride(u1), 0)
}
