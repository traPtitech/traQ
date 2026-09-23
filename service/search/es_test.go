package search

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewESEngineCancelsInitialization(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := NewESEngine(ctx, nil, nil, nil, zap.NewNop(), ESEngineConfig{URL: server.URL})
		result <- err
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("Elasticsearch initialization did not start")
	}

	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Elasticsearch initialization did not stop after cancellation")
	}
}
