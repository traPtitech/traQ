package oidc

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/utils"
)

type Service struct {
	repo   repository.Repository
	origin string
	rbac   rbac.RBAC
}

func NewOIDCService(
	repo repository.Repository,
	origin string,
	rbac rbac.RBAC,
) *Service {
	return &Service{
		repo:   repo,
		origin: origin,
		rbac:   rbac,
	}
}

type ScopeChecker interface {
	Contains(scope model.AccessScope) bool
}

func (s *Service) GetUserInfo(userID uuid.UUID, scopes ScopeChecker) (map[string]any, error) {
	user, err := s.repo.GetUser(userID, true)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.GetUserTagsByUserID(user.GetID())
	if err != nil {
		return nil, err
	}
	groups, err := s.repo.GetUserBelongingGroupIDs(user.GetID())
	if err != nil {
		return nil, err
	}

	// Build claims
	claims := make(map[string]any)

	// Required in UserInfo response
	claims["sub"] = userID.String()

	// Also supply some basic traQ ID related information
	// These claims usually belong to 'profile' scope according to OpenID spec,
	// but in traQ, we will just supply traQ ID related information as well
	claims["name"] = user.GetName()
	claims["preferred_username"] = user.GetName()
	claims["picture"] = s.origin + "/api/v3/public/icon/" + user.GetName()

	// Scope specific claims
	if scopes.Contains("profile") {
		claims["updated_at"] = user.GetUpdatedAt().Unix()
		// Putting non-standard traq-specific claims under the 'traq.' key
		// for clarity and to avoid possible conflict with other standard claims
		claims["traq"] = map[string]any{
			"bio":          user.GetBio(),
			"groups":       groups,
			"tags":         utils.Map(tags, func(tag model.UserTag) string { return tag.GetTag() }),
			"last_online":  user.GetLastOnline(),
			"twitter_id":   user.GetTwitterID(),
			"display_name": user.GetResponseDisplayName(),
			"icon_file_id": user.GetIconFileID(),
			"bot":          user.IsBot(),
			"state":        user.GetState().Int(),
			"permissions":  s.rbac.GetGrantedPermissions(user.GetRole()),
			"home_channel": user.GetHomeChannel(),
		}
	}

	return claims, nil
}
