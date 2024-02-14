package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fetchVRChatWorldInfo(t *testing.T) {
	tests := []struct {
		name    string
		worldID string
		want    func(t *testing.T, res *VRChatAPIWorldResponse)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "success",
			worldID: "wrld_aa762efb-17b3-4302-8f41-09c4db2489ed",
			want: func(t *testing.T, res *VRChatAPIWorldResponse) {
				assert.Equal(t, "PROJECT˸ SUMMER FLARE", res.Name)
				assert.Equal(t, "Break the Summer․ Break the tower․ Complete the Meridian Loop․ A story about reality and humanity․", res.Description)
				assert.True(t, strings.HasPrefix(res.ImageURL, "https://"))
				assert.True(t, strings.HasPrefix(res.ThumbnailImageURL, "https://"))
			},
			wantErr: assert.NoError,
		},
		{
			name:    "not found",
			worldID: "wrld_aa762efb-17b3-4302-8f41-09c4db2489ee",
			want: func(_ *testing.T, _ *VRChatAPIWorldResponse) {
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, ErrClient, i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchVRChatWorldInfo(tt.worldID)
			if !tt.wantErr(t, err, fmt.Sprintf("fetchVRChatWorldInfo(%v)", tt.worldID)) {
				return
			}
			tt.want(t, got)
		})
	}
}
