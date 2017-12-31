package model

import (
	"fmt"
	"testing"
)

var (
	password = "test"
)

func TestSetPassword(t *testing.T) {
	beforeTest(t)
	user := &User{
		Name:  "testUser",
		Email: "hogehoge@gmail.com",
		Icon:  "po",
	}

	if err := user.SetPassword(password); err != nil {
		t.Fatalf("Failed to setPassword: %v", err)
	}

	if err := user.Create(); err != nil {
		t.Fatalf("Failed to user Create: %v", err)
	}

	if err := checkEmptyField(user); err != nil {
		t.Fatal(err)
	}

	hashedPassword := hashPassword(password, user.Salt)

	if hashedPassword != user.Password {
		t.Fatal("password not match")
	}

}

func TestAuthorization(t *testing.T) {
	beforeTest(t)
	user := &User{
		Name:  "testUser",
		Email: "hogehoge@gmail.com",
		Icon:  "po",
	}

	if err := user.SetPassword(password); err != nil {
		t.Fatalf("Failed to setPassword: %v", err)
	}

	if err := user.Create(); err != nil {
		t.Fatalf("Failed to user Create: %v", err)
	}

	checkUser := &User{
		Name: "testUser",
	}

	err := checkUser.Authorization(password)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := checkEmptyField(checkUser); err != nil {
		t.Fatalf("some checkUser params are empty: %v", err)
	}
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
