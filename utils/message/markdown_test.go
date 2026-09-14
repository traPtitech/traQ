package message

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func TestRustMarkdown(t *testing.T) {
	id := "ee764d5f-71d9-4a40-bc7b-547d8d097c91"
	reference := `!{"type":"user","raw":"@test","id":"` + id + `"}`
	result, err := Parse(context.Background(), "**hello** !!secret!!\n`"+reference+"`\n"+reference)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{uuid.Must(uuid.FromString(id))}, result.Mentions)
	require.Contains(t, result.NotificationText(), "hello ██████")
	require.NotContains(t, result.NotificationText(), "secret")
	require.NotContains(t, result.NotificationText(), "\n")
	// The bot/search payload retains its existing formatting and attachment contract.
	require.Contains(t, result.PlainText, "**hello** !!secret!!")
}

func TestRustMarkdownRecoversAfterFailure(t *testing.T) {
	_, err := Parse(context.Background(), strings.Repeat("x", 65537))
	require.Error(t, err)
	result, err := Parse(context.Background(), "**next request**")
	require.NoError(t, err)
	require.Equal(t, "next request", result.NotificationText())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Parse(ctx, "cancelled")
	require.True(t, errors.Is(err, context.Canceled))
	result, err = Parse(context.Background(), "still available")
	require.NoError(t, err)
	require.Equal(t, "still available", result.NotificationText())
}
