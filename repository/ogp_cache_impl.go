package repository

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"reflect"
	"time"
)

const cacheHours = 7 * 24

func getCacheExpireDate() time.Time {
	return time.Now().Add(time.Duration(cacheHours) * time.Hour)
}


// CreateOgpCache implements OgpRepository interface.
func (repo *GormRepository)CreateOgpCache(url string, content model.Ogp) (c *model.OgpCache, err error) {
	ogpCache := &model.OgpCache{
		URL:          url,
		Content:      content,
		ExpiresAt:    getCacheExpireDate(),
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(ogpCache).Error
	})
	if err != nil {
		return nil, err
	}
	return ogpCache, nil
}

// UpdateOgpCache implements OgpRepository interface.
func (repo *GormRepository)UpdateOgpCache(url string, content model.Ogp) error {
	changes := map[string]interface{}{}
	return repo.db.Transaction(func(tx *gorm.DB) error {
		var c model.OgpCache
		if err := tx.First(&c, &model.OgpCache{ URL: url }).Error; err != nil {
			return convertError(err)
		}

		if !reflect.DeepEqual(c.Content, content) {
			changes["content"] = content
		}
		if len(changes) > 0 {
			changes["expires_at"] = getCacheExpireDate()
			return tx.Model(&c).Updates(changes).Error
		}
		return nil
	})
}

// GetOgpCache implements OgpRepository interface.
func (repo *GormRepository)GetOgpCache(url string) (c *model.OgpCache, err error) {
	c = &model.OgpCache{}
	if err = repo.db.Take(c, &model.OgpCache{URL: url}).Error; err != nil {
		return nil, convertError(err)
	}
	return c, nil
}

// DeleteOgpCache implements OgpRepository interface.
func (repo *GormRepository)DeleteOgpCache(url string) error {
	c, err := repo.GetOgpCache(url)
	if err != nil {
		return err
	}
	result := repo.db.Delete(c)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
