package validator

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestNotNilUUID(t *testing.T) {
	t.Parallel()

	t.Run("ok (nil)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(nil))
	})
	t.Run("ok (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))))
	})
	t.Run("ok (uuid.NullUUID)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(optional.From(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")))))
	})
	t.Run("ok (uuid.NullUUID Valid:false)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(optional.Of[uuid.UUID]{}))
	})
	t.Run("ok (string)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate("550e8400-e29b-41d4-a716-446655440000"))
	})
	t.Run("ok ([]byte)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")).Bytes()))
	})
	t.Run("ng (int)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(1))
	})
	t.Run("ng (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil))
	})
	t.Run("ng (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(optional.From(uuid.Nil)))
	})
	t.Run("ng (string)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil.String()))
	})
	t.Run("ng ([]byte)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil.Bytes()))
	})
}

func TestOAuth2RedirectURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"nil", nil, false},
		{"empty", "", false},
		{"https", "https://example.com/callback", false},
		{"https with query", "https://example.com/callback?foo=bar", false},
		{"http loopback ipv4", "http://127.0.0.1:12345/callback", false},
		{"http loopback ipv6", "http://[::1]/callback", false},
		{"private-use scheme", "com.example.app:/oauth2redirect/example-provider", false},
		{"private-use scheme with authority", "jp.trap.qall://callback", false},
		{"private-use scheme with ip literal", "jp.trap.qall://[::1]/callback", false},
		{"private-use scheme with rootless path", "com.example.app:callback", false},
		{"private-use scheme with empty path and query", "com.example.app:?foo=bar", false},
		{"private-use scheme with empty path", "com.example.app:", false},
		{"private-use scheme uppercase", "Com.Example.App:/callback", false},
		{"private-use scheme with symbols", "com.example-app.v2+x:/callback", false},
		{"optional private-use scheme", optional.From("com.example.app:/callback"), false},
		{"optional not valid", optional.Of[string]{}, false},
		{"optional invalid", optional.From("javascript:alert(1)"), true},
		{"non reverse domain scheme", "myapp:/callback", true},
		{"non reverse domain scheme with authority", "myapp://callback", true},
		{"scheme ending with dot", "com.:/callback", true},
		{"scheme with empty label", "com..example:/callback", true},
		{"private-use scheme with fragment", "com.example.app:/callback#foo", true},
		{"private-use scheme with backslash", `com.example.app:/a\b`, true},
		{"private-use scheme with invalid percent-encoding", "com.example.app:%zz", true},
		{"private-use scheme with space", "jp.trap.qall:/cb x", true},
		{"private-use scheme with non-ascii", "com.example.app:/caf\u00e9", true},
		{"https with backslash", `https://example.com\callback`, true},
		{"https with invalid percent-encoding", "https://example.com/%zz", true},
		{"javascript", "javascript:alert(1)", true},
		{"data", "data:text/html,<script>alert(1)</script>", true},
		{"file", "file:///etc/passwd", true},
		{"invalid", "not a url", true},
		{"int", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := OAuth2RedirectURI.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
