package repository

import (
	"crypto/sha1"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
	"reflect"
	"time"
)

const cacheHours = 7 * 24

func getCacheExpireDate() time.Time {
	return time.Now().Add(time.Duration(cacheHours) * time.Hour)
}

func getURLHash(url string) (string, error) {
	hash := sha1.New()
	_, err := hash.Write([]byte(url))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CreateOgpCache implements OgpRepository interface.
func (repo *GormRepository)CreateOgpCache(url string, content model.Ogp) (c *model.OgpCache, err error) {
	urlHash, err := getURLHash(url)
	if err != nil {
		return nil, err
	}

	ogpCache := &model.OgpCache{
		URL:          url,
		URLHash:      urlHash,
		Valid:        true,
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

// CreateOgpCacheNegative implements OgpRepository interface.
func (repo *GormRepository)CreateOgpCacheNegative(url string) (c *model.OgpCache, err error) {
	urlHash, err := getURLHash(url)
	if err != nil {
		return nil, err
	}

	ogpCache := &model.OgpCache{
		URL:          url,
		URLHash:      urlHash,
		Valid:        false,
		Content:      model.Ogp{},
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
	urlHash, err := getURLHash(url)
	if err != nil {
		return err
	}

	changes := map[string]interface{}{}
	return repo.db.Transaction(func(tx *gorm.DB) error {
		var c model.OgpCache
		if err := tx.First(&c, &model.OgpCache{ URL: url, URLHash: urlHash }).Error; err != nil {
			return convertError(err)
		}

		if !reflect.DeepEqual(c.Content, content) {
			changes["valid"] = true
			changes["content"] = content
			changes["expires_at"] = getCacheExpireDate()
			return tx.Model(&c).Updates(changes).Error
		}
		return nil
	})
}

// UpdateOgpCacheNegative implements OgpRepository interface.
func (repo *GormRepository)UpdateOgpCacheNegative(url string) error {
	urlHash, err := getURLHash(url)
	if err != nil {
		return err
	}

	changes := map[string]interface{}{}
	return repo.db.Transaction(func(tx *gorm.DB) error {
		var c model.OgpCache
		if err := tx.First(&c, &model.OgpCache{ URL: url, URLHash: urlHash }).Error; err != nil {
			return convertError(err)
		}

		if c.Valid == true {
			changes["valid"] = false
			changes["content"] = model.Ogp{}
			changes["expires_at"] = getCacheExpireDate()
			return tx.Model(&c).Updates(changes).Error
		}
		return nil
	})
}

// GetOgpCache implements OgpRepository interface.
func (repo *GormRepository)GetOgpCache(url string) (c *model.OgpCache, err error) {
	urlHash, err := getURLHash(url)
	if err != nil {
		return nil, err
	}

	c = &model.OgpCache{}
	if err = repo.db.Take(c, &model.OgpCache{ URL: url, URLHash: urlHash }).Error; err != nil {
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
