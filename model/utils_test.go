package model

import "testing"

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
