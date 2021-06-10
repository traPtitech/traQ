package rbac

import (
	"fmt"
	"sync"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
)

type rbacImpl struct {
	roles      role.Roles
	rolesMutex sync.RWMutex
	repo       repository.Repository
}

// New RBACを初期化
func New(repo repository.Repository) (RBAC, error) {
	rbac := &rbacImpl{
		roles: role.Roles{},
		repo:  repo,
	}
	if err := rbac.reload(); err != nil {
		return nil, fmt.Errorf("failed to init rbac: %w", err)
	}
	return rbac, nil
}

func (r *rbacImpl) IsGranted(role string, perm permission.Permission) bool {
	r.rolesMutex.RLock()
	defer r.rolesMutex.RUnlock()
	return r.isGranted(role, perm)
}

func (r *rbacImpl) isGranted(_role string, p permission.Permission) bool {
	if _role == role.Admin {
		return true
	}
	return r.roles.HasAndIsGranted(_role, p)
}

func (r *rbacImpl) IsAllGranted(roles []string, perm permission.Permission) bool {
	r.rolesMutex.RLock()
	defer r.rolesMutex.RUnlock()
	for _, role := range roles {
		if !r.isGranted(role, perm) {
			return false
		}
	}
	return true
}

func (r *rbacImpl) IsAnyGranted(roles []string, perm permission.Permission) bool {
	r.rolesMutex.RLock()
	defer r.rolesMutex.RUnlock()
	for _, role := range roles {
		if r.isGranted(role, perm) {
			return true
		}
	}
	return false
}

func (r *rbacImpl) Reload() error {
	return r.reload()
}

func (r *rbacImpl) reload() error {
	rs, err := r.repo.GetAllUserRoles()
	if err != nil {
		return err
	}

	roles := map[string]*roleImpl{}
	roleMap := map[string]*model.UserRole{}
	for _, v := range rs {
		roleMap[v.Name] = v

		perms := permission.Permissions{}
		for _, v := range v.Permissions {
			perms.Add(permission.Permission(v.Permission))
		}

		roles[v.Name] = &roleImpl{
			name:         v.Name,
			oauth2:       v.Oauth2Scope,
			inheritances: role.Roles{},
			permissions:  perms,
		}
	}

	for _, v := range roleMap {
		p := roles[v.Name]
		for _, i := range v.Inheritances {
			p.inheritances.Add(roles[i.Name])
		}
	}

	// TODO 木の循環検知

	result := role.Roles{}
	for _, v := range roles {
		result.Add(v)
	}
	r.rolesMutex.Lock()
	r.roles = result
	r.rolesMutex.Unlock()
	return nil
}

func (r *rbacImpl) GetGrantedPermissions(roleName string) []permission.Permission {
	if roleName == role.Admin {
		return permission.List
	}
	r.rolesMutex.RLock()
	ro, ok := r.roles[roleName]
	r.rolesMutex.RUnlock()
	if ok {
		return ro.Permissions().Array()
	}
	return nil
}

type roleImpl struct {
	name         string
	oauth2       bool
	inheritances role.Roles
	permissions  permission.Permissions
}

func (r *roleImpl) Name() string {
	return r.name
}

func (r *roleImpl) IsGranted(p permission.Permission) bool {
	return r.permissions.Contains(p) || r.inheritances.IsGranted(p)
}

func (r *roleImpl) Permissions() permission.Permissions {
	result := permission.Permissions{}
	for k := range r.permissions {
		result.Add(k)
	}
	for _, v := range r.inheritances {
		for k := range v.Permissions() {
			result.Add(k)
		}
	}
	return result
}
