package ogpparser

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/utils/imaging"
)

func TestIsPrivateIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"fd00::1", true},
		{"169.254.169.254", true},
		{"fe80::1", true},
		{"0.0.0.0", true},
		{"0.1.2.3", true},
		{"::", true},
		{"224.0.0.1", true},
		{"100.64.0.1", true},
		{"100.100.100.200", true},
		{"198.18.0.1", true},
		{"255.255.255.255", true},
		{"::ffff:127.0.0.1", true},
		{"::ffff:169.254.169.254", true},
		{"64:ff9b::7f00:1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2001:4860:4860::8888", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			t.Parallel()
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tt.want, isPrivateIP(ip))
		})
	}
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isPrivateIP(nil))
	})
}

func TestClientBlocksPrivateIP(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(testHTML))
	}))
	t.Cleanup(server.Close)

	t.Run("ParseMetaForURL", func(t *testing.T) {
		t.Parallel()
		u, err := url.Parse(server.URL)
		require.NoError(t, err)
		_, _, err = ParseMetaForURL(u)
		assert.ErrorIs(t, err, ErrNetwork)
	})
	t.Run("FetchImageSize", func(t *testing.T) {
		t.Parallel()
		_, _, err := imaging.FetchImageSize(context.Background(), &client, server.URL+"/image.png")
		assert.ErrorContains(t, err, "private IP address is not allowed")
	})
}
