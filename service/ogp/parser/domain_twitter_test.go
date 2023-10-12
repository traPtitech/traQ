package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fetchTwitterSyndicationAPI(t *testing.T) {
	// Twitter APIが使えなくなってしまったため、関数が使えないためスキップ
	t.SkipNow()

	tests := []struct {
		name     string
		statusID string
		want     func(t *testing.T, res *TwitterSyndicationAPIResponse)
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "success",
			statusID: "990696508403040256",
			want: func(t *testing.T, res *TwitterSyndicationAPIResponse) {
				assert.Equal(t, "@_winnie_on は?情緒不安定か?ﾏﾝｺﾞｰうまいぜ", res.Text)
			},
			wantErr: assert.NoError,
		},
		{
			name:     "not found",
			statusID: "990696508403040257",
			want: func(t *testing.T, res *TwitterSyndicationAPIResponse) {
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, ErrClient, i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchTwitterSyndicationAPI(tt.statusID)
			if !tt.wantErr(t, err, fmt.Sprintf("fetchTwitterSyndicationAPI(%v)", tt.statusID)) {
				return
			}
			tt.want(t, got)
		})
	}
}
