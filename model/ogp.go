package model

import (
	"database/sql/driver"
	"errors"
	"time"

	"github.com/traPtitech/traQ/utils/optional"
)

// OgpMedia OGPの画像・映像に関する情報の構造体
type OgpMedia struct {
	URL       string              `json:"url"`
	SecureURL optional.Of[string] `json:"secureUrl"`
	Type      optional.Of[string] `json:"type"`
	Width     optional.Of[int]    `json:"width"`
	Height    optional.Of[int]    `json:"height"`
}

// Ogp OGP情報の構造体
type Ogp struct {
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Images      []OgpMedia `json:"images"`
	Description string     `json:"description"`
	Videos      []OgpMedia `json:"videos"`
}

// OgpCache Ogpのキャッシュ情報
type OgpCache struct {
	ID        int       `gorm:"auto_increment;not null;primaryKey"`
	URL       string    `gorm:"type:text;not null"`
	URLHash   string    `gorm:"type:char(40);not null;index"`
	Valid     bool      `gorm:"type:boolean"`
	Content   Ogp       `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"precision:6"`
}

// TableName OGPキャッシュデータのテーブル名
func (ogp *OgpCache) TableName() string {
	return "ogp_cache"
}

// Value database/sql/driver.Valuer 実装
func (o Ogp) Value() (driver.Value, error) {
	return json.MarshalToString(o)
}

// Scan database/sql.Scanner 実装
func (o *Ogp) Scan(src interface{}) error {
	*o = Ogp{}
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		return json.Unmarshal([]byte(s), o)
	case []byte:
		return json.Unmarshal(s, o)
	default:
		return errors.New("failed to scan Ogp")
	}
}
