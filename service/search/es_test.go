package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/message"
)

func TestESEngine_Do_From(t *testing.T) {
	t.Parallel()

	user1 := uuid.Must(uuid.NewV4())
	user2 := uuid.Must(uuid.NewV4())
	user3 := uuid.Must(uuid.NewV4())
	group1 := uuid.Must(uuid.NewV4())
	group2 := uuid.Must(uuid.NewV4())
	emptyGroup := uuid.Must(uuid.NewV4())
	largeGroup := uuid.Must(uuid.NewV4())
	unknownID := uuid.Must(uuid.NewV4())
	groups := map[uuid.UUID]*model.UserGroup{
		group1: {
			ID: group1,
			Members: []*model.UserGroupMember{
				{UserID: user1}, {UserID: user2},
			},
		},
		group2: {
			ID: group2,
			Members: []*model.UserGroupMember{
				{UserID: user2}, {UserID: user3},
			},
		},
		emptyGroup: {ID: emptyGroup},
		largeGroup: {ID: largeGroup},
	}
	largeGroupUsers := make([]uuid.UUID, 1500)
	for i := range largeGroupUsers {
		largeGroupUsers[i] = uuid.Must(uuid.NewV4())
		groups[largeGroup].Members = append(groups[largeGroup].Members, &model.UserGroupMember{UserID: largeGroupUsers[i]})
	}

	for _, tc := range []struct {
		name      string
		from      []uuid.UUID
		wantUsers []uuid.UUID
	}{
		{name: "no from filter"},
		{name: "user", from: []uuid.UUID{user1}, wantUsers: []uuid.UUID{user1}},
		{name: "multiple users", from: []uuid.UUID{user1, user2}, wantUsers: []uuid.UUID{user1, user2}},
		{name: "group", from: []uuid.UUID{group1}, wantUsers: []uuid.UUID{user1, user2}},
		{name: "overlapping groups", from: []uuid.UUID{group1, group2}, wantUsers: []uuid.UUID{user1, user2, user3}},
		{name: "user and group", from: []uuid.UUID{user3, group1}, wantUsers: []uuid.UUID{user3, user1, user2}},
		{name: "user also in group", from: []uuid.UUID{user1, group1}, wantUsers: []uuid.UUID{user1, user2}},
		{name: "empty group", from: []uuid.UUID{emptyGroup}},
		{name: "empty group and user", from: []uuid.UUID{emptyGroup, user1}, wantUsers: []uuid.UUID{user1}},
		{name: "unknown ID", from: []uuid.UUID{unknownID}, wantUsers: []uuid.UUID{unknownID}},
		{name: "duplicate IDs", from: []uuid.UUID{group1, group1, user1, user1}, wantUsers: []uuid.UUID{user1, user2}},
		{name: "large group", from: []uuid.UUID{largeGroup}, wantUsers: largeGroupUsers},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &searchTestRepository{groups: groups, calls: make(map[uuid.UUID]int)}
			var requestBody string
			client, err := elasticsearch.NewClient(elasticsearch.Config{
				Transport: searchTestTransport(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "/traq_message/_search", req.URL.Path)
					body, err := io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					requestBody = string(body)
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
						Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0},"hits":[]}}`)),
					}, nil
				}),
			})
			require.NoError(t, err)
			engine := &esEngine{client: client, repo: repo, mm: searchTestMessageManager{}, l: zap.NewNop()}
			query := &Query{From: tc.from}

			result, err := engine.Do(query)
			require.NoError(t, err)
			assert.Zero(t, result.TotalHits())
			assert.Empty(t, result.Hits())
			assert.Equal(t, tc.from, query.From, "search must not mutate the input query")

			wantFrom := ""
			if len(tc.from) > 0 {
				if len(tc.wantUsers) == 0 {
					wantFrom = `,{"match_none":{}}`
				} else {
					usersJSON, err := json.Marshal(tc.wantUsers)
					require.NoError(t, err)
					wantFrom = fmt.Sprintf(`,{"terms":{"userId":%s}}`, usersJSON)
				}
			}
			assert.JSONEq(t, fmt.Sprintf(`{"query":{"bool":{"must":[{"term":{"isPublic":{"value":true}}}%s]}}}`, wantFrom), requestBody)
			for _, id := range tc.from {
				assert.Equal(t, 1, repo.calls[id], "each distinct ID should be resolved once")
			}
			if len(tc.from) == 0 {
				assert.Empty(t, repo.calls)
			}
		})
	}
}

func TestESEngine_Do_FromGroupError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	engine := &esEngine{
		repo: &searchTestRepository{err: wantErr},
		l:    zap.NewNop(),
	}
	result, err := engine.Do(&Query{From: []uuid.UUID{uuid.Must(uuid.NewV4())}})
	assert.Nil(t, result)
	assert.ErrorIs(t, err, wantErr)
}

type searchTestRepository struct {
	repository.Repository
	groups map[uuid.UUID]*model.UserGroup
	calls  map[uuid.UUID]int
	err    error
}

func (r *searchTestRepository) GetUserGroup(_ context.Context, id uuid.UUID) (*model.UserGroup, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.calls[id]++
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, fmt.Errorf("group lookup: %w", repository.ErrNotFound)
}

type searchTestMessageManager struct {
	message.Manager
}

func (searchTestMessageManager) GetIn(context.Context, []uuid.UUID) ([]message.Message, error) {
	return nil, nil
}

type searchTestTransport func(*http.Request) (*http.Response, error)

func (f searchTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
