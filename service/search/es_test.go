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
	"github.com/samber/lo"
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
		{name: "many source IDs", from: largeGroupUsers, wantUsers: largeGroupUsers},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &searchTestRepository{groups: groups}
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
			engine := &esEngine{client: client, repo: repo, mm: &searchTestMessageManager{}, l: zap.NewNop()}
			query := &Query{From: tc.from}

			result, err := engine.Do(context.Background(), query)
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
			if len(tc.from) == 0 {
				assert.Zero(t, repo.calls)
			} else {
				assert.Equal(t, 1, repo.calls, "all source IDs should be resolved in one batch")
				assert.ElementsMatch(t, lo.Uniq(tc.from), repo.requestedIDs)
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
	result, err := engine.Do(context.Background(), &Query{From: []uuid.UUID{uuid.Must(uuid.NewV4())}})
	assert.Nil(t, result)
	assert.ErrorIs(t, err, wantErr)
}

func TestESEngine_Do_Context(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	for _, cancelAt := range []string{"none", "group lookup", "elasticsearch", "message loading"} {
		t.Run(cancelAt, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "search request"))
			defer cancel()
			repo := &searchTestRepository{}
			manager := &searchTestMessageManager{}
			var searchCtx context.Context
			client, err := elasticsearch.NewClient(elasticsearch.Config{
				DisableRetry: true,
				Transport: searchTestTransport(func(req *http.Request) (*http.Response, error) {
					searchCtx = req.Context()
					if cancelAt == "elasticsearch" {
						cancel()
						assert.ErrorIs(t, searchCtx.Err(), context.Canceled)
						return nil, context.Canceled
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
						Body:       io.NopCloser(strings.NewReader(`{"hits":{"total":{"value":0},"hits":[]}}`)),
					}, nil
				}),
			})
			require.NoError(t, err)
			engine := &esEngine{client: client, repo: repo, mm: manager, l: zap.NewNop()}
			switch cancelAt {
			case "group lookup":
				cancel()
			case "message loading":
				manager.beforeGet = cancel
			}

			result, err := engine.Do(ctx, &Query{From: []uuid.UUID{uuid.Must(uuid.NewV4())}})
			if cancelAt == "none" {
				require.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				assert.ErrorIs(t, err, context.Canceled)
				assert.Nil(t, result)
			}

			require.NotNil(t, repo.ctx)
			assert.Equal(t, "search request", repo.ctx.Value(contextKey{}))
			if cancelAt == "group lookup" {
				assert.Nil(t, searchCtx, "cancelled group lookups must stop before Elasticsearch")
			} else {
				require.NotNil(t, searchCtx)
				assert.Equal(t, "search request", searchCtx.Value(contextKey{}))
			}
			if cancelAt == "group lookup" || cancelAt == "elasticsearch" {
				assert.Nil(t, manager.ctx, "failed searches must stop before message loading")
			} else {
				require.NotNil(t, manager.ctx)
				assert.Equal(t, "search request", manager.ctx.Value(contextKey{}))
			}
		})
	}
}

type searchTestRepository struct {
	repository.Repository
	groups       map[uuid.UUID]*model.UserGroup
	calls        int
	requestedIDs []uuid.UUID
	ctx          context.Context
	err          error
}

func (r *searchTestRepository) GetUserGroupMemberIDs(ctx context.Context, groupIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	r.calls++
	r.requestedIDs = groupIDs
	r.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	members := make(map[uuid.UUID][]uuid.UUID)
	for _, id := range groupIDs {
		if group, ok := r.groups[id]; ok {
			members[id] = []uuid.UUID{}
			for _, member := range group.Members {
				members[id] = append(members[id], member.UserID)
			}
		}
	}
	return members, nil
}

type searchTestMessageManager struct {
	message.Manager
	ctx       context.Context
	beforeGet func()
}

func (m *searchTestMessageManager) GetIn(ctx context.Context, _ []uuid.UUID) ([]message.Message, error) {
	m.ctx = ctx
	if m.beforeGet != nil {
		m.beforeGet()
	}
	return nil, ctx.Err()
}

type searchTestTransport func(*http.Request) (*http.Response, error)

func (f searchTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
