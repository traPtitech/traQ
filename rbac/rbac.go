package rbac

import (
	"github.com/gofrs/uuid"
	"github.com/mikespook/gorbac"
	"sync"
)

// RBAC : Role-Based Access Controller extending gorbac
type RBAC struct {
	*gorbac.RBAC
	mutex     sync.RWMutex
	store     Store
	overrides map[uuid.UUID]map[gorbac.Permission]bool
}

// New : 空のRBACを生成します
func New(store Store) (*RBAC, error) {
	rbac := &RBAC{
		RBAC:      gorbac.New(),
		overrides: make(map[uuid.UUID]map[gorbac.Permission]bool),
	}

	// Restore
	if store != nil {
		overrides, err := store.GetAllOverrides()
		if err != nil {
			return nil, err
		}
		for _, v := range overrides {
			_ = rbac.SetOverride(v.GetUserID(), v.GetPermission(), v.GetValidity())
		}

		rbac.store = store
	}

	return rbac, nil
}

// IsGranted tests if the role `ID` has Permission `p`. it may be overridden according to userID.
func (rbac *RBAC) IsGranted(userID uuid.UUID, roleID string, p gorbac.Permission) bool {
	rbac.mutex.RLock()
	defer rbac.mutex.RUnlock()

	override, ok := rbac.overrides[userID]
	if ok {
		if state, ok := override[p]; ok {
			return state
		}
	}

	return rbac.RBAC.IsGranted(roleID, p, nil)
}

// GetOverride : 指定したユーザーに付与されているオーバライドルールを取得します
func (rbac *RBAC) GetOverride(userID uuid.UUID) (result map[gorbac.Permission]bool) {
	rbac.mutex.RLock()
	if override, ok := rbac.overrides[userID]; ok {
		// Copy
		result = map[gorbac.Permission]bool{}
		for k, v := range override {
			result[k] = v
		}
	}
	rbac.mutex.RUnlock()
	return
}

// SetOverride : 指定したユーザーにオーバーライドルールを付与します
func (rbac *RBAC) SetOverride(userID uuid.UUID, p gorbac.Permission, validity bool) (err error) {
	rbac.mutex.Lock()
	if override, ok := rbac.overrides[userID]; ok {
		override[p] = validity
		rbac.overrides[userID] = override
	} else {
		override = make(map[gorbac.Permission]bool)
		override[p] = validity
		rbac.overrides[userID] = override
	}

	// Persistence
	if rbac.store != nil {
		err = rbac.store.SaveOverride(userID, p, validity)
	}

	rbac.mutex.Unlock()
	return
}

// DeleteOverride : 指定したユーザーの指定したパーミッションのオーバーライドルールを削除します
func (rbac *RBAC) DeleteOverride(userID uuid.UUID, p gorbac.Permission) (err error) {
	rbac.mutex.Lock()
	if override, ok := rbac.overrides[userID]; ok {
		delete(override, p)

		// Persistence
		if rbac.store != nil {
			err = rbac.store.DeleteOverride(userID, p)
		}
	}
	rbac.mutex.Unlock()
	return
}
