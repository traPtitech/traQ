package model

import (
	"fmt"
	"reflect"
	"testing"
)

var (
	password = "test"
)

func TestSetPassword(t *testing.T) {
	beforeTest(t)
	user, err := createUser("testUser")
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
	user, err := createUser("testGetUser")
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
	_, err := createUser("testUser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	checkUser := &User{
		Name: "testUser",
	}

	err = checkUser.Authorization(password)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := checkEmptyField(checkUser); err != nil {
		t.Fatalf("some checkUser params are empty: %v", err)
	}
}

func createUser(userName string) (*User, error) {
	user := &User{
		Name:  userName,
		Email: "hogehoge@gmail.com",
		Icon:  "po",
	}

	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("Failed to setPassword: %v", err)
	}
	if err := user.Create(); err != nil {
		return nil, fmt.Errorf("Failed to user Create: %v", err)
	}

	return user, nil
}

func checkEmptyField(user *User) error {
	if user.ID == "" {
		return fmt.Errorf("ID is empty")
	}
	if user.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if user.Email == "" {
		return fmt.Errorf("Email is empty")
	}
	if user.Password == "" {
		return fmt.Errorf("Password is empty")
	}
	if user.Salt == "" {
		return fmt.Errorf("Salt is empty")
	}
	if user.Icon == "" {
		return fmt.Errorf("Icon is empty")
	}
	if user.Status == 0 {
		return fmt.Errorf("Status is empty")
	}
	return nil
}
