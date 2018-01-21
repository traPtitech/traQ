package model

import (
	"reflect"
	"testing"
)

var (
	password = "test"
)

func TestSetPassword(t *testing.T) {
	beforeTest(t)
	user, err := makeUser("testUser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if err := checkEmptyField(user); err != nil {
		t.Fatal(err)
	}

	hashedPassword := hashPassword(password, user.Salt)

	if hashedPassword != user.Password {
		t.Fatal("password not match")
	}

}

func TestGetUser(t *testing.T) {
	beforeTest(t)
	// 正常系
	user, err := makeUser("testGetUser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	getUser, err := GetUser(user.ID)
	if err != nil {
		t.Fatalf("an error occurred in GetUser : %v", err)
	}

	// DB格納時に記録されるデータをコピー
	user.CreatedAt = getUser.CreatedAt
	user.UpdatedAt = getUser.UpdatedAt

	if !reflect.DeepEqual(user, getUser) {
		t.Fatal("some fields are changed while getting user from database")
	}

	// 異常系
	notExistID := CreateUUID()
	if _, err := GetUser(notExistID); err == nil {
		t.Fatalf("GetUser doesn't throw an error: Following userID doesn't exist: %v", notExistID)
	}
}

func TestAuthorization(t *testing.T) {
	beforeTest(t)
	_, err := makeUser("testUser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	checkUser := &User{
		Name: "testUser",
	}

	if err := checkUser.Authorization(password); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := checkEmptyField(checkUser); err != nil {
		t.Fatalf("some checkUser params are empty: %v", err)
	}
}
