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

func TestIsPrivateUseURIScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scheme string
		want   bool
	}{
		{"com.example.app", true},
		{"jp.trap.qall", true},
		{"Com.Example.App", true},
		{"com.example-app.v2+x", true},
		{"myapp", false},
		{"http", false},
		{"https", false},
		{"javascript", false},
		{"data", false},
		{"file", false},
		{"com.", false},
		{".com", false},
		{"com..example", false},
		{"1com.example", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsPrivateUseURIScheme(tt.scheme))
		})
	}
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
		{"private-use scheme with double slash", "jp.trap.qall://callback", false},
		{"private-use scheme with opaque", "com.example.app:callback", false},
		{"private-use scheme uppercase", "Com.Example.App:/callback", false},
		{"optional private-use scheme", optional.From("com.example.app:/callback"), false},
		{"optional invalid", optional.From("javascript:alert(1)"), true},
		{"optional not valid", optional.Of[string]{}, false},
		{"fragment", "https://example.com/callback#foo", true},
		{"private-use scheme with fragment", "com.example.app:/callback#foo", true},
		{"private-use scheme without path", "com.example.app:", true},
		{"non reverse domain scheme", "myapp:/callback", true},
		{"non reverse domain scheme with double slash", "myapp://callback", true},
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
