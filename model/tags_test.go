package model

import (
	"testing"
)

func TestTagTableName(t *testing.T) {
	correctName := "tags"
	tag := &Tag{}

	if tag.TableName() != correctName {
		t.Errorf("Table name is wrong. want: %s, actual: %s", correctName, tag.TableName())
	}
}

func TestCreateTag(t *testing.T) {
	beforeTest(t)

	// 正常系
	tag := &Tag{
		Name: "Create test",
	}
	if err := tag.Create(); err != nil {
		t.Fatal(err)
	}

	var dbTag = &Tag{}
	has, err := db.Get(dbTag)

	if err != nil {
		t.Errorf("Failed to get tag: %v", err)
	}
	if !has {
		t.Error("Cannot find tag in DB")
	}
	if dbTag.Name != tag.Name {
		t.Errorf("Name is wrong. want: %s, actual: %s", tag.Name, dbTag.Name)
	}

	// 異常系
	wrongTag := &Tag{}
	if err := wrongTag.Create(); err == nil {
		t.Error("no error for invalid request")
	}
}

func TestExistsTag(t *testing.T) {
	beforeTest(t)

	// 正常系
	tag := &Tag{
		Name: "existTag",
	}
	if err := tag.Create(); err != nil {
		t.Fatal(err)
	}

	has, err := tag.Exists()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("missing tag: %v", tag)
	}

	tag = &Tag{
		ID:   CreateUUID(),
		Name: "wrong tag",
	}

	has, err = tag.Exists()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("This tag shouldn't exist. but something is found.")
	}
}

func TestGetTagByID(t *testing.T) {
	beforeTest(t)

	// 正常系
	tag := &Tag{
		Name: "getTag",
	}
	if err := tag.Create(); err != nil {
		t.Fatal(err)
	}
	gotTag, err := GetTagByID(tag.ID)
	if err != nil {
		t.Fatal(err)
	}

	if gotTag.Name != tag.Name {
		t.Errorf("Tag name doesn't match. want: %s, actual: %s", tag.Name, gotTag.Name)
	}

	if _, err := GetTagByID(CreateUUID()); err == nil {
		t.Error("no error for invalid request")
	}
}
