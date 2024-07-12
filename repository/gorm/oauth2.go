package gorm

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/random"
)

// GetClient implements OAuth2Repository interface.
func (repo *Repository) GetClient(id string) (*model.OAuth2Client, error) {
	if len(id) == 0 {
		return nil, repository.ErrNotFound
	}
	oc := &model.OAuth2Client{}
	if err := repo.db.Take(oc, &model.OAuth2Client{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return oc, nil
}

// GetClients implements OAuth2Repository interface.
func (repo *Repository) GetClients(query repository.GetClientsQuery) ([]*model.OAuth2Client, error) {
	cs := make([]*model.OAuth2Client, 0)
	tx := repo.db
	if query.DeveloperID.Valid {
		tx = tx.Where("creator_id = ?", query.DeveloperID.V)
	}
	return cs, tx.Find(&cs).Error
}

// SaveClient implements OAuth2Repository interface.
func (repo *Repository) SaveClient(client *model.OAuth2Client) error {
	return repo.db.Create(client).Error
}

// UpdateClient implements OAuth2Repository interface.
func (repo *Repository) UpdateClient(clientID string, args repository.UpdateClientArgs) error {
	if len(clientID) == 0 {
		return repository.ErrNilID
	}
	return repo.db.Transaction(func(tx *gorm.DB) error {
		var oc model.OAuth2Client
		if err := repo.db.Where(&model.OAuth2Client{ID: clientID}).First(&oc).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Name.Valid {
			changes["name"] = args.Name.V
		}
		if args.Description.Valid {
			changes["description"] = args.Description.V
		}
		if args.Secret.Valid {
			changes["secret"] = args.Secret.V
		}
		if args.Confidential.Valid {
			changes["confidential"] = args.Confidential.V
		}
		if args.Scopes != nil {
			changes["scopes"] = args.Scopes
		}
		if args.CallbackURL.Valid {
			changes["redirect_uri"] = args.CallbackURL.V
		}
		if args.DeveloperID.Valid {
			// 作成者検証
			user, err := repo.GetUser(args.DeveloperID.V, false)
			if err != nil {
				if err == repository.ErrNotFound {
					return repository.ArgError("args.DeveloperID", "the Developer is not found")
				}
				return err
			}
			if !user.IsActive() || user.IsBot() {
				return repository.ArgError("args.DeveloperID", "invalid User")
			}

			changes["creator_id"] = args.DeveloperID.V
		}

		if len(changes) > 0 {
			if err := tx.Model(&oc).Updates(changes).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteClient implements OAuth2Repository interface.
func (repo *Repository) DeleteClient(id string) error {
	if len(id) == 0 {
		return nil
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.OAuth2Client{ID: id}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.OAuth2Authorize{}, &model.OAuth2Authorize{ClientID: id}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.OAuth2Token{}, &model.OAuth2Token{ClientID: id}).Error
	})
	return err
}

// SaveAuthorize implements OAuth2Repository interface.
func (repo *Repository) SaveAuthorize(data *model.OAuth2Authorize) error {
	return repo.db.Create(data).Error
}

// GetAuthorize implements OAuth2Repository interface.
func (repo *Repository) GetAuthorize(code string) (*model.OAuth2Authorize, error) {
	if len(code) == 0 {
		return nil, repository.ErrNotFound
	}
	oa := &model.OAuth2Authorize{}
	if err := repo.db.Take(oa, &model.OAuth2Authorize{Code: code}).Error; err != nil {
		return nil, convertError(err)
	}
	return oa, nil
}

// DeleteAuthorize implements OAuth2Repository interface.
func (repo *Repository) DeleteAuthorize(code string) error {
	if len(code) == 0 {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Authorize{Code: code}).Error
}

// IssueToken implements OAuth2Repository interface.
func (repo *Repository) IssueToken(client *model.OAuth2Client, userID uuid.UUID, redirectURI string, scope model.AccessScopes, expire int, refresh bool) (*model.OAuth2Token, error) {
	newToken := &model.OAuth2Token{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         userID,
		RedirectURI:    redirectURI,
		AccessToken:    random.SecureAlphaNumeric(36),
		RefreshToken:   random.SecureAlphaNumeric(36),
		RefreshEnabled: refresh,
		CreatedAt:      time.Now(),
		ExpiresIn:      expire,
		Scopes:         scope,
	}

	if client != nil {
		newToken.ClientID = client.ID
	}

	return newToken, repo.db.Create(newToken).Error
}

// GetTokenByID implements OAuth2Repository interface.
func (repo *Repository) GetTokenByID(id uuid.UUID) (*model.OAuth2Token, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	ot := &model.OAuth2Token{}
	if err := repo.db.Take(ot, &model.OAuth2Token{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return ot, nil
}

// DeleteTokenByID implements OAuth2Repository interface.
func (repo *Repository) DeleteTokenByID(id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{ID: id}).Error
}

// GetTokenByAccess implements OAuth2Repository interface.
func (repo *Repository) GetTokenByAccess(access string) (*model.OAuth2Token, error) {
	if len(access) == 0 {
		return nil, repository.ErrNotFound
	}
	ot := &model.OAuth2Token{}
	if err := repo.db.Take(ot, &model.OAuth2Token{AccessToken: access}).Error; err != nil {
		return nil, convertError(err)
	}
	return ot, nil
}

// DeleteTokenByAccess implements OAuth2Repository interface.
func (repo *Repository) DeleteTokenByAccess(access string) error {
	if len(access) == 0 {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{AccessToken: access}).Error
}

// GetTokenByRefresh implements OAuth2Repository interface.
func (repo *Repository) GetTokenByRefresh(refresh string) (*model.OAuth2Token, error) {
	if len(refresh) == 0 {
		return nil, repository.ErrNotFound
	}
	ot := &model.OAuth2Token{}
	if err := repo.db.Take(ot, &model.OAuth2Token{RefreshToken: refresh, RefreshEnabled: true}).Error; err != nil {
		return nil, convertError(err)
	}
	return ot, nil
}

// DeleteTokenByRefresh implements OAuth2Repository interface.
func (repo *Repository) DeleteTokenByRefresh(refresh string) error {
	if len(refresh) == 0 {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{RefreshToken: refresh, RefreshEnabled: true}).Error
}

// GetTokensByUser implements OAuth2Repository interface.
func (repo *Repository) GetTokensByUser(userID uuid.UUID) ([]*model.OAuth2Token, error) {
	ts := make([]*model.OAuth2Token, 0)
	if userID == uuid.Nil {
		return ts, nil
	}
	return ts, repo.db.Where(&model.OAuth2Token{UserID: userID}).Find(&ts).Error
}

// DeleteTokenByUser implements OAuth2Repository interface.
func (repo *Repository) DeleteTokenByUser(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{UserID: userID}).Error
}

// DeleteTokenByClient implements OAuth2Repository interface.
func (repo *Repository) DeleteTokenByClient(clientID string) error {
	if len(clientID) == 0 {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{ClientID: clientID}).Error
}

func (repo *Repository) DeleteUserTokensByClient(userID uuid.UUID, clientID string) error {
	if userID == uuid.Nil || len(clientID) == 0 {
		return nil
	}
	return repo.db.Delete(&model.OAuth2Token{}, &model.OAuth2Token{UserID: userID, ClientID: clientID}).Error
}
