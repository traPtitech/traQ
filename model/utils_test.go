package model

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("MARIADB_DATABASE", "traq-test-model")
	code := m.Run()
	os.Exit(code)
}

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
