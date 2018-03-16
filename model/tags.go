package model

// Tag tag_idの管理をする構造体
type Tag struct {
	ID   string `xorm:"char(36) pk"`
	Name string `xorm:"varchar(30) not null unique"`
}

// TableName DBの名前を指定
func (*Tag) TableName() string {
	return "tags"
}

// Create DBにタグを追加
func (t *Tag) Create() error {
	if t.Name == "" {
		return ErrInvalidParam
	}

	t.ID = CreateUUID()

	if _, err := db.Insert(t); err != nil {
		return err
	}
	return nil
}

// Exists DBにその名前のタグが存在するかを確認
func (t *Tag) Exists() (bool, error) {
	if t.Name == "" {
		return false, ErrInvalidParam
	}
	return db.Get(t)
}

// GetTagByID 引数のIDを持つTag構造体を返す
func GetTagByID(ID string) (*Tag, error) {
	tag := &Tag{
		ID: ID,
	}

	has, err := db.Get(tag)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}
	return tag, nil
}

// GetAllTags 全てのタグを取得する
func GetAllTags() (result []*Tag, err error) {
	err = db.Find(&result)
	return
}
