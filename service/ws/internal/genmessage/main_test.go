package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedMessagesAreCurrent(t *testing.T) {
	t.Parallel()

	generated, err := generate("../../../../docs/v3-api.yaml")
	require.NoError(t, err)
	current, err := os.ReadFile("../../messages_gen.go")
	require.NoError(t, err)
	require.Equal(t, string(generated), string(bytes.ReplaceAll(current, []byte("\r\n"), []byte("\n"))),
		"run go generate ./service/ws after changing the WebSocket schemas")
}
