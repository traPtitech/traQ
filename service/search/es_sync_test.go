package search

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func TestEsMessageMapping_ImageVector(t *testing.T) {
	t.Parallel()

	properties := esMessageMapping()["properties"].(m)

	if _, ok := properties["imageText"]; ok {
		t.Fatal("imageText must not be mapped")
	}

	imageVector, ok := properties["imageVector"].(m)
	if !ok {
		t.Fatal("imageVector must be mapped")
	}
	if imageVector["type"] != "nested" {
		t.Fatalf("unexpected imageVector type: %v", imageVector["type"])
	}

	nested := imageVector["properties"].(m)
	vector := nested["vector"].(m)
	if vector["type"] != "dense_vector" {
		t.Fatalf("unexpected vector type: %v", vector["type"])
	}
	if vector["dims"] != EmbeddingDimension {
		t.Fatalf("unexpected vector dims: %v", vector["dims"])
	}
	if vector["similarity"] != "cosine" {
		t.Fatalf("unexpected vector similarity: %v", vector["similarity"])
	}
	for _, field := range []string{"jobId", "fileId"} {
		if nested[field].(m)["type"] != "keyword" {
			t.Fatalf("%s must be keyword", field)
		}
	}
}

func TestIsEmbeddableImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta *model.FileMeta
		want bool
	}{
		{"png", &model.FileMeta{Mime: "image/png", Size: 1}, true},
		{"jpeg", &model.FileMeta{Mime: "image/jpeg", Size: 1}, true},
		{"webp", &model.FileMeta{Mime: "image/webp", Size: 1}, true},
		{"gif", &model.FileMeta{Mime: "image/gif", Size: 1}, false},
		{"animated", &model.FileMeta{Mime: "image/png", Size: 1, IsAnimatedImage: true}, false},
		{"too-large", &model.FileMeta{Mime: "image/png", Size: maxEmbeddingImageSize + 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmbeddableImage(tt.meta); got != tt.want {
				t.Fatalf("isEmbeddableImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestImageJobRepo(jobs ...*model.ImageEmbeddingJob) *testImageJobRepository {
	repo := &testImageJobRepository{jobs: make(map[uuid.UUID]*model.ImageEmbeddingJob, len(jobs))}
	for _, job := range jobs {
		repo.jobs[job.JobID] = job
	}
	return repo
}

func newTestESClient(t *testing.T, handler searchTestTransport) *elasticsearch.Client {
	t.Helper()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Transport: handler})
	require.NoError(t, err)
	return client
}

func esTestResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestESEngine_CompleteImageJob_DocumentMissing(t *testing.T) {
	t.Parallel()

	jobID := uuid.Must(uuid.NewV4())
	messageID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())
	repo := newTestImageJobRepo(&model.ImageEmbeddingJob{
		JobID:     jobID,
		MessageID: messageID,
		FileID:    fileID,
		Status:    model.ImageEmbeddingJobProcessing,
	})
	client := newTestESClient(t, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/traq_message/_update/"+messageID.String(), req.URL.Path)
		return esTestResponse(http.StatusNotFound, `{"error":{"type":"document_missing_exception","reason":"[traq_message][id]: document missing"},"status":404}`), nil
	})
	engine := &esEngine{client: client, repo: repo, l: zap.NewNop()}

	err := engine.completeImageJob(context.Background(), jobID, []float64{0.1, 0.2})
	require.NoError(t, err)
	assert.Equal(t, model.ImageEmbeddingJobCompleted, repo.jobs[jobID].Status)
}

func TestESEngine_CompleteImageJob_ESUpdateError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{"index not found", http.StatusNotFound, `{"error":{"type":"index_not_found_exception"},"status":404}`},
		{"server error", http.StatusInternalServerError, `{"error":{"type":"internal_server_error"},"status":500}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			jobID := uuid.Must(uuid.NewV4())
			repo := newTestImageJobRepo(&model.ImageEmbeddingJob{
				JobID:     jobID,
				MessageID: uuid.Must(uuid.NewV4()),
				FileID:    uuid.Must(uuid.NewV4()),
				Status:    model.ImageEmbeddingJobProcessing,
			})
			client := newTestESClient(t, func(*http.Request) (*http.Response, error) {
				return esTestResponse(tc.statusCode, tc.body), nil
			})
			engine := &esEngine{client: client, repo: repo, l: zap.NewNop()}

			err := engine.completeImageJob(context.Background(), jobID, []float64{0.1, 0.2})
			require.Error(t, err)
			assert.Equal(t, model.ImageEmbeddingJobProcessing, repo.jobs[jobID].Status)
		})
	}
}

func TestESEngine_ReconcileImageJobs_FailsAfterTimeout(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		createdAt time.Time
		want      model.ImageEmbeddingJobStatus
	}{
		{"timed out job is marked failed", time.Now().Add(-7 * time.Hour), model.ImageEmbeddingJobFailed},
		{"recent job stays pending", time.Now(), model.ImageEmbeddingJobPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			jobID := uuid.Must(uuid.NewV4())
			repo := newTestImageJobRepo(&model.ImageEmbeddingJob{
				JobID:     jobID,
				MessageID: uuid.Must(uuid.NewV4()),
				FileID:    uuid.Must(uuid.NewV4()),
				Status:    model.ImageEmbeddingJobPending,
				CreatedAt: tc.createdAt,
			})
			client := newTestESClient(t, func(*http.Request) (*http.Response, error) {
				return esTestResponse(http.StatusInternalServerError, `{"error":{"type":"internal_server_error"},"status":500}`), nil
			})
			engine := &esEngine{
				client: client,
				repo:   repo,
				l:      zap.NewNop(),
				imageClient: &testImageJobClient{
					status: ImageJobCompleted,
					vector: []float64{0.1, 0.2},
				},
				imageConfig: ImageSearchConfig{JobTimeout: 6 * time.Hour},
			}

			require.NoError(t, engine.reconcileImageJobs(context.Background()))
			assert.Equal(t, tc.want, repo.jobs[jobID].Status)
		})
	}
}

type testImageJobRepository struct {
	repository.Repository
	jobs map[uuid.UUID]*model.ImageEmbeddingJob
}

func (r *testImageJobRepository) GetImageEmbeddingJob(_ context.Context, jobID uuid.UUID) (*model.ImageEmbeddingJob, error) {
	job, ok := r.jobs[jobID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return job, nil
}

func (r *testImageJobRepository) UpdateImageEmbeddingJobStatus(_ context.Context, jobID uuid.UUID, status model.ImageEmbeddingJobStatus) error {
	job, ok := r.jobs[jobID]
	if !ok {
		return repository.ErrNotFound
	}
	job.Status = status
	return nil
}

func (r *testImageJobRepository) GetUnfinishedImageEmbeddingJobs(_ context.Context, limit int) ([]*model.ImageEmbeddingJob, error) {
	jobs := make([]*model.ImageEmbeddingJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		if !job.Status.IsFinished() {
			jobs = append(jobs, job)
		}
	}
	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

type testImageJobClient struct {
	status ImageJobStatus
	vector []float64
}

func (c *testImageJobClient) EmbedText(context.Context, string) ([]float64, error) {
	return c.vector, nil
}

func (c *testImageJobClient) SubmitImageJob(context.Context, string, string, io.Reader, string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (c *testImageJobClient) GetJob(context.Context, uuid.UUID) (ImageJobStatus, []float64, error) {
	return c.status, c.vector, nil
}

func (c *testImageJobClient) Available() bool {
	return true
}
