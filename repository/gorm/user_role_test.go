package gorm

import (
	"sort"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGormRepository_CreateUserRoles(t *testing.T) {
	t.Parallel()

	repo, assert, _ := setup(t, common)

	r1 := &model.UserRole{Name: "r1", Permissions: []model.RolePermission{{Permission: "p1"}}}
	r2 := &model.UserRole{Name: "r2", Permissions: []model.RolePermission{{Permission: "p2"}}}
	r3 := &model.UserRole{Name: "r3", Permissions: []model.RolePermission{{Permission: "p3"}}}
	r4 := &model.UserRole{Name: "r4", Permissions: []model.RolePermission{{Permission: "p4"}}}

	r2.Inheritances = append(r2.Inheritances, r3)
	r3.Inheritances = append(r3.Inheritances, r4)

	err := repo.CreateUserRoles(r1, r2, r3, r4)
	assert.NoError(err)
}

func TestGormRepository_GetAllUserRoles(t *testing.T) {
	t.Parallel()

	repo, assert, require := setup(t, common2)

	r1 := &model.UserRole{Name: "r1", Permissions: []model.RolePermission{{Permission: "p1"}}}
	r2 := &model.UserRole{Name: "r2", Permissions: []model.RolePermission{{Permission: "p2"}}}
	r3 := &model.UserRole{Name: "r3", Permissions: []model.RolePermission{{Permission: "p3"}}}
	r4 := &model.UserRole{Name: "r4", Permissions: []model.RolePermission{{Permission: "p4"}}}

	r2.Inheritances = append(r2.Inheritances, r3)
	r3.Inheritances = append(r3.Inheritances, r4)

	err := repo.CreateUserRoles(r1, r2, r3, r4)
	require.NoError(err)

	roles, err := repo.GetAllUserRoles()
	if assert.NoError(err) {
		assert.Len(roles, 4)
		sort.Slice(roles, func(i, j int) bool {
			return roles[i].Name < roles[j].Name
		})
		if assert.EqualValues(roles[0].Name, r1.Name) {
			assert.Len(roles[0].Inheritances, 0)
		}
		if assert.EqualValues(roles[1].Name, r2.Name) {
			if assert.Len(roles[1].Inheritances, 1) {
				assert.EqualValues(roles[1].Inheritances[0].Name, r3.Name)
			}
		}
		if assert.EqualValues(roles[2].Name, r3.Name) {
			if assert.Len(roles[2].Inheritances, 1) {
				assert.EqualValues(roles[2].Inheritances[0].Name, r4.Name)
			}
		}
		if assert.EqualValues(roles[3].Name, r4.Name) {
			assert.Len(roles[3].Inheritances, 0)
		}
	}
}
