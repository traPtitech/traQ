package model

import "testing"

var (
	testUserID    = "403807a5-cae6-453e-8a09-fc75d5b4ca91"
	privateUserID = "8ad765ec-426b-49c1-b4ae-f8af58af9a55"
)

func TestDB(t *testing.T) {
	err := EstablishConnection()
	if err != nil {
		t.Fatal("Failed to EstablishConnection\n", err)
	}

	err = Close()
	if err != nil {
		t.Fatal("Failed to Disconnect\n", err)
	}
}

func TestSyncSchema(t *testing.T) {
	err := EstablishConnection()
	defer Close()
	if err != nil {
		t.Fatal("Failed to EstablishConnection\n", err)
	}

	err = SyncSchema()
	if err != nil {
		t.Fatal("Failed to SyncSchema\n", err)
	}
}
