package model

import (
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
		Tag:    "全強",
	}
	if err := tag.Create(); err != nil {
		t.Fatalf("Create method returned an error: %v", err)
	}

	var dbTag = &UsersTag{}
	has, err := db.Get(dbTag)
	if !has {
		t.Error("Cannot find tag in DB")
	}
	if err != nil {
		t.Errorf("Failed to get tag: %v", err)
	}

	if tag.ID != dbTag.ID {
		t.Errorf("ID is wrong. want: %s, actual: %s", tag.ID, dbTag.ID)
	}
	if tag.Tag != dbTag.Tag {
		t.Errorf("Tag is wrong. want: %s, actual: %s", tag.Tag, dbTag.Tag)
	}
	if dbTag.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}

	// 異常系
	wrongTag := &UsersTag{}
	if err := wrongTag.Create(); err == nil {
		t.Error("no error for bad request")
	}
}

func TestUpdateTag(t *testing.T) {
	beforeTest(t)

	tag := &UsersTag{
		UserID: testUserID,
		Tag:    "pro",
	}
	if err := tag.Create(); err != nil {
		t.Fatalf("create method returned an error: %v", err)
	}

	tag.IsLocked = true
	if err := tag.Update(); err != nil {
		t.Fatalf("update method returned an error: %v", err)
	}

	var dbTag = &UsersTag{}
	has, err := db.Get(dbTag)
	if !has {
		t.Error("Cannot find tag in DB")
	}
	if err != nil {
		t.Errorf("Failed to get tag: %v", err)
	}

	if dbTag.IsLocked != true {
		t.Error("IsLocked is not updated")
	}
	if dbTag.UpdatedAt == tag.UpdatedAt {
		t.Error("updatedAt is not updated")
	}
}

func TestGetTagsByUserID(t *testing.T) {
	beforeTest(t)

	// 正常系
	var tags [10]*UsersTag
	for i := 0; i < len(tags); i++ {
		tags[i] = &UsersTag{
			UserID: testUserID,
			Tag:    CreateUUID(),
		}
		if err := tags[i].Create(); err != nil {
			t.Fatalf("Failed to create tag: %v", err)
		}
	}

	gotTags, err := GetTagsByUserID(testUserID)
	if err != nil {
		t.Errorf("Failed to get tags from userID: %v", err)
	}
	for i, v := range gotTags {
		if v.ID != tags[i].ID {
			t.Errorf("ID is wrong. want: %s, actual: %s", tags[i].ID, v.ID)
		}
	}

	// 異常系
	notExistID := CreateUUID()
	empty, err := GetTagsByUserID(notExistID)
	if err != nil {
		t.Errorf("GetTagsByID returned an error for no exist ID request: %v", err)
	}
	if len(empty) != 0 {
		t.Error("no Tags should found, but some tags found")
	}
}

func TestGetTag(t *testing.T) {
	beforeTest(t)

	tagText := "test"
	// 正常系
	tag := &UsersTag{
		UserID: testUserID,
		Tag:    tagText,
	}
	if err := tag.Create(); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	getTag, err := GetTag(tag.ID)
	if err != nil {
		t.Fatalf("Failed to get tag by ID: %v", err)
	}

	if getTag.UserID != tag.UserID {
		t.Errorf("UserID is wrong. want: %s, actual: %s", getTag.UserID, tag.UserID)
	}
	if getTag.Tag != tag.Tag {
		t.Errorf("Tag is wrong. want: %s, actual: %s", getTag.Tag, tag.Tag)
	}

	// 異常系
	wrongTagID := CreateUUID()
	if _, err := GetTag(wrongTagID); err == nil {
		t.Error("no error for bad request")
	}
}
