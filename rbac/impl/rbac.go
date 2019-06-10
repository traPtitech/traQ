package impl

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	systemRole "github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"sync"
	"time"
)

type rbacImpl struct {
	roles      rbac.Roles
	rolesMutex sync.RWMutex
	repo       repository.Repository
	reloadTime time.Time
}

// New RBACを初期化
func New(repo repository.Repository) (rbac.RBAC, error) {
	rbac := &rbacImpl{
		roles: rbac.Roles{},
		repo:  repo,
	}
	if err := rbac.Reload(); err != nil {
		return nil, err
	}
	return rbac, nil
}

func (r *rbacImpl) IsGranted(role string, perm rbac.Permission) bool {
	r.rolesMutex.RLock()
	ok := r.isGranted(role, perm)
	r.rolesMutex.RUnlock()
	return ok
}

func (r *rbacImpl) isGranted(role string, p rbac.Permission) bool {
	if role == systemRole.Admin {
		return true
	}
	return r.roles.HasAndIsGranted(role, p)
}

func (r *rbacImpl) IsAllGranted(roles []string, perm rbac.Permission) bool {
	ok := true
	r.rolesMutex.RLock()
	for _, role := range roles {
		if !r.isGranted(role, perm) {
			ok = false
			break
		}
	}
	r.rolesMutex.RUnlock()
	return ok
}

func (r *rbacImpl) IsAnyGranted(roles []string, perm rbac.Permission) bool {
	ok := false
	r.rolesMutex.RLock()
	for _, role := range roles {
		if r.isGranted(role, perm) {
			ok = true
			break
		}
	}
	r.rolesMutex.RUnlock()
	return ok
}

func (r *rbacImpl) Reload() error {
	rs, err := r.repo.GetAllRoles()
	if err != nil {
		return err
	}

	roles := map[string]*role{}
	roleMap := map[string]*model.UserRole{}
	for _, v := range rs {
		roleMap[v.Name] = v

		perms := rbac.Permissions{}
		for _, v := range v.Permissions {
			perms.Add(rbac.Permission(v.Permission))
		}

		roles[v.Name] = &role{
			name:         v.Name,
			oauth2:       v.Oauth2Scope,
			inheritances: rbac.Roles{},
			permissions:  perms,
		}
	}

	for _, v := range roleMap {
		p := roles[v.Name]
		for _, i := range v.Inheritances {
			p.inheritances.Add(roles[i.SubRole])
		}
	}

	// TODO 木の循環検知

	result := rbac.Roles{}
	for _, v := range roles {
		result.Add(v)
	}
	r.rolesMutex.Lock()
	r.roles = result
	r.reloadTime = time.Now()
	r.rolesMutex.Unlock()
	return nil
}

func (r *rbacImpl) LastReloadTime() time.Time {
	return r.reloadTime
}

func (r *rbacImpl) IsOAuth2Scope(v string) bool {
	r.rolesMutex.RLock()
	role, ok := r.roles[v]
	r.rolesMutex.RUnlock()
	return ok && role.IsOAuth2Scope()
}

func (r *rbacImpl) IsValidRole(v string) bool {
	r.rolesMutex.RLock()
	_, ok := r.roles[v]
	r.rolesMutex.RUnlock()
	return ok
}

func (r *rbacImpl) GetGrantedPermissions(roleName string) rbac.Permissions {
	r.rolesMutex.RLock()
	ro, ok := r.roles[roleName]
	r.rolesMutex.RUnlock()
	if ok {
		return ro.Permissions()
	}
	return nil
}
