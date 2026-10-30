package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

func testVector() []float64 {
	v := make([]float64, EmbeddingDimension)
	for i := range v {
		v[i] = float64(i) / float64(EmbeddingDimension)
	}
	return v
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (ImageSearchClient, *httptest.Server) {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return NewImageSearchClient(ImageSearchConfig{
		URL:             s.URL,
		APIKey:          "test-key",
		TextTimeout:     5 * time.Second,
		JobPollInterval: 10 * time.Millisecond,
		JobTimeout:      time.Minute,
	}, zap.NewNop()), s
}

func TestImageSearchClient_EmbedText(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings/text" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected authorization: %s", got)
		}
		var req struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Text != "猫" {
			t.Errorf("unexpected text: %s", req.Text)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"vector": testVector()})
	})

	vector, err := client.EmbedText(context.Background(), "  猫  ")
	if err != nil {
		t.Fatalf("EmbedText returned error: %v", err)
	}
	if len(vector) != EmbeddingDimension {
		t.Fatalf("unexpected vector length: %d", len(vector))
	}
}

func TestImageSearchClient_EmbedTextRetriesOn503(t *testing.T) {
	t.Parallel()

	var calls int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"vector": testVector()})
	})

	if _, err := client.EmbedText(context.Background(), "cat"); err != nil {
		t.Fatalf("EmbedText returned error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("unexpected call count: %d", got)
	}
}

func TestImageSearchClient_EmbedTextReturnsErrorOn504(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	})

	if _, err := client.EmbedText(context.Background(), "cat"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestImageSearchClient_SubmitImageJob(t *testing.T) {
	t.Parallel()

	jobID := uuid.Must(uuid.NewV4())
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings/images" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
		}
		file, _, err := r.FormFile("images")
		if err != nil {
			t.Errorf("missing images field: %v", err)
		} else {
			defer file.Close()
			body, _ := io.ReadAll(file)
			if string(body) != "image-bytes" {
				t.Errorf("unexpected image body: %s", body)
			}
		}
		if got := r.FormValue("webhook_url"); got != "https://example.com/webhook" {
			t.Errorf("unexpected webhook_url: %s", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": jobID.String()})
	})

	got, err := client.SubmitImageJob(context.Background(), "image.png", "image/png", strings.NewReader("image-bytes"), "https://example.com/webhook")
	if err != nil {
		t.Fatalf("SubmitImageJob returned error: %v", err)
	}
	if got != jobID {
		t.Fatalf("unexpected job id: %s", got)
	}
}

func TestImageSearchClient_GetJob(t *testing.T) {
	t.Parallel()

	jobID := uuid.Must(uuid.NewV4())
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings/jobs/"+jobID.String() {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     jobID.String(),
			"status": "completed",
			"result": map[string]any{"vector": testVector()},
		})
	})

	status, vector, err := client.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetJob returned error: %v", err)
	}
	if status != ImageJobCompleted {
		t.Fatalf("unexpected status: %s", status)
	}
	if len(vector) != EmbeddingDimension {
		t.Fatalf("unexpected vector length: %d", len(vector))
	}
}

func TestImageSearchClient_Available(t *testing.T) {
	t.Parallel()

	if NewImageSearchClient(ImageSearchConfig{}, zap.NewNop()).Available() {
		t.Fatal("client without URL must be unavailable")
	}
	if !NewImageSearchClient(ImageSearchConfig{URL: "http://localhost"}, zap.NewNop()).Available() {
		t.Fatal("client with URL must be available")
	}
}
