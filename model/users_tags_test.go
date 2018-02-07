package model

import (
	"strconv"
	"testing"
)

func TestUserTagTableName(t *testing.T) {
	tag := &UsersTag{}
	correctName := "users_tags"
	tableName := tag.TableName()
	if tableName != correctName {
		t.Errorf("tag's table name is wrong. want: %s, actual: %s", correctName, tableName)
	}
}

func TestCreateUserTag(t *testing.T) {
	beforeTest(t)

	// 正常系
	tag := &UsersTag{
		UserID: testUserID,
	}
	if err := tag.Create("全強"); err != nil {
		t.Fatal(err)
	}

	var dbTag = &UsersTag{}
	has, err := db.Get(dbTag)
	if !has {
		t.Error("Cannot find tag in DB")
	}
	if err != nil {
		t.Error(err)
	}

	if tag.TagID != dbTag.TagID {
		t.Errorf("TagID is wrong. want: %s, actual: %s", tag.TagID, dbTag.TagID)
	}
	if tag.UserID != dbTag.UserID {
		t.Errorf("UserID is wrong. want: %s, actual: %s", tag.UserID, dbTag.UserID)
	}
	if dbTag.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}

	// 異常系
	wrongTag := &UsersTag{}
	if err := wrongTag.Create("po"); err == nil {
		t.Error("no error for bad request")
	}
}

func TestUpdateTag(t *testing.T) {
	beforeTest(t)

	tag := &UsersTag{
		UserID: testUserID,
	}
	if err := tag.Create("pro"); err != nil {
		t.Fatal(err)
	}

	tag.IsLocked = true
	if err := tag.Update(); err != nil {
		t.Fatal(err)
	}

	var dbTag = &UsersTag{}
	has, err := db.Get(dbTag)
	if !has {
		t.Error("Cannot find tag in DB")
	}
	if err != nil {
		t.Error(err)
	}

	if dbTag.IsLocked != true {
		t.Error("IsLocked is not updated")
	}
	if dbTag.UpdatedAt == tag.UpdatedAt {
		t.Error("updatedAt is not updated")
	}
}

func TestGetUserTagsByUserID(t *testing.T) {
	beforeTest(t)

	// 正常系
	var tags [10]*UsersTag
	for i := 0; i < len(tags); i++ {
		tags[i] = &UsersTag{
			UserID: testUserID,
		}
		if err := tags[i].Create(strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}

	gotTags, err := GetUserTagsByUserID(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range gotTags {
		if v.TagID != tags[i].TagID {
			t.Errorf("ID is wrong. want: %s, actual: %s", tags[i].TagID, v.TagID)
		}
	}

	// 異常系
	notExistID := CreateUUID()
	empty, err := GetUserTagsByUserID(notExistID)
	if err != nil {
		t.Error(err)
	}
	if len(empty) != 0 {
		t.Error("no Tags should be found, but some tags are found")
	}
}

func TestGetTag(t *testing.T) {
	beforeTest(t)

	tagText := "test"
	// 正常系
	tag := &UsersTag{
		UserID: testUserID,
	}
	if err := tag.Create(tagText); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	getTag, err := GetTag(tag.UserID, tag.TagID)
	if err != nil {
		t.Fatal(err)
	}

	if getTag.UserID != tag.UserID {
		t.Errorf("UserID is wrong. want: %s, actual: %s", getTag.UserID, tag.UserID)
	}
	if getTag.TagID != tag.TagID {
		t.Errorf("TagID is wrong. want: %s, actual: %s", getTag.TagID, tag.TagID)
	}

	// 異常系
	wrongTagID := CreateUUID()
	if _, err := GetTag(testUserID, wrongTagID); err == nil {
		t.Error("no error for bad request")
	}
}
