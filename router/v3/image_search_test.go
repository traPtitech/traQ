package v3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/service/search"
)

type fakeImageWebhookEngine struct {
	search.Engine
	calledID uuid.UUID
	err      error
}

func (f *fakeImageWebhookEngine) HandleImageWebhook(_ context.Context, jobID uuid.UUID) error {
	f.calledID = jobID
	return f.err
}

func TestHandlers_ImageSearchWebhook(t *testing.T) {
	t.Parallel()

	jobID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name       string
		body       string
		engineErr  error
		wantStatus int
		wantCalled bool
	}{
		{"completed", `{"id":"` + jobID.String() + `","status":"completed"}`, nil, http.StatusNoContent, true},
		{"failed", `{"id":"` + jobID.String() + `","status":"failed"}`, nil, http.StatusNoContent, true},
		{"engine error is ignored", `{"id":"` + jobID.String() + `","status":"completed"}`, errors.New("boom"), http.StatusNoContent, true},
		{"invalid id", `{"id":"00000000-0000-0000-0000-000000000000","status":"completed"}`, nil, http.StatusBadRequest, false},
		{"invalid status", `{"id":"` + jobID.String() + `","status":"pending"}`, nil, http.StatusBadRequest, false},
		{"malformed json", `{`, nil, http.StatusBadRequest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := &fakeImageWebhookEngine{err: tt.engineErr}
			h := &Handlers{SearchEngine: engine, Logger: zap.NewNop()}

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v3/search/image-webhook", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.ImageSearchWebhook(c)
			if tt.wantStatus == http.StatusNoContent {
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, rec.Code)
			} else {
				require.Error(t, err)
				var he *echo.HTTPError
				require.ErrorAs(t, err, &he)
				assert.Equal(t, tt.wantStatus, he.Code)
			}
			assert.Equal(t, tt.wantCalled, engine.calledID == jobID)
		})
	}
}
