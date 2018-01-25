package model

// Tag userTagの構造体
type Tag struct {
	UserID    string `xorm:"char(36) pk"`
	Tag       string `xorm:"text pk"`
	IsLocked  bool   `xorm:"bool not null"`
	CreatedAt string `xorm:"created not null"`
	UpdatedAt string `xorm:"updated not null"`
}

// TableName DBの名前を指定
func (tag *Tag) TableName() string {
	return "users_tags"
}

// Create DBに新規タグを追加します
func (tag *Tag) Create() error {
	return nil
}

// Update データの更新をします
func (tag *Tag) Update() error {
	return nil
}

// Delete データを消去します。正しく消せた場合はレシーバはnilになります
func (tag *Tag) Delete() error {
	return nil
}

// GetTagsByID userIDに紐づくtagのリストを返します
func GetTagsByID(userID string) ([]*Tag, error) {
	return nil, nil
}

// GetTag userIDとtagで一意に定まるタグを返します
func GetTag(userID, tag string) (*Tag, error) {
	return nil, nil
}
