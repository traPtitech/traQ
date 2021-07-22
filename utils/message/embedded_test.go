package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEmbedding(t *testing.T) {
	t.Parallel()

	type Case struct {
		Message string
		Plain   string
		Infos   []EmbeddedInfo
	}

	cases := []Case{
		{
			"test message !{aaa",
			"test message !{aaa",
			[]EmbeddedInfo{},
		},
		{
			`{"test": "test"}!!{}`,
			`{"test": "test"}!!{}`,
			[]EmbeddedInfo{},
		},
		{
			`!{aiueo::::aaaaaaa}`,
			`!{aiueo::::aaaaaaa}`,
			[]EmbeddedInfo{},
		},
		{
			`test message !{"test": "test"}`,
			`test message !{"test": "test"}`,
			[]EmbeddedInfo{},
		},
		{
			`test message !{"raw": "@test","type":"user","id":"test_id"}`,
			`test message @test`,
			[]EmbeddedInfo{
				{
					Raw:  "@test",
					Type: "user",
					ID:   "test_id",
				},
			},
		},
		{
			`!{"raw": "","type":"file","id":"aaaa"}test message !{"raw": "@test","type":"user","id":"test_id"}`,
			`[添付ファイル]test message @test`,
			[]EmbeddedInfo{
				{
					Raw:  "@test",
					Type: "user",
					ID:   "test_id",
				},
				{
					Raw:  "",
					Type: "file",
					ID:   "aaaa",
				},
			},
		},
		{
			`!{ test message !{"raw": "@test","type":"user","id":"test_id"}`,
			`!{ test message @test`,
			[]EmbeddedInfo{
				{
					Raw:  "@test",
					Type: "user",
					ID:   "test_id",
				},
			},
		},
		{
			`!{ test message !{"raw": 1,"type":"user","id":"test_id"}`,
			`!{ test message !{"raw": 1,"type":"user","id":"test_id"}`,
			[]EmbeddedInfo{},
		},
		{
			`!{ test message !{"raw": 1,"type":"","id":"test_id"}`,
			`!{ test message !{"raw": 1,"type":"","id":"test_id"}`,
			[]EmbeddedInfo{},
		},
	}

	for _, v := range cases {
		v := v
		t.Run(v.Plain, func(t *testing.T) {
			t.Parallel()
			res, plain := ExtractEmbedding(v.Message)
			deref := make([]EmbeddedInfo, len(res))
			for k, v := range res {
				deref[k] = *v
			}
			assert.EqualValues(t, v.Plain, plain)
			assert.ElementsMatch(t, v.Infos, deref)
		})
	}
}
