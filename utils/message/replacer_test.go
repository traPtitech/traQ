package message

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestReplaceMapper struct {
	ChannelMap map[string]uuid.UUID
	UserMap    map[string]uuid.UUID
	GroupMap   map[string]uuid.UUID
}

func (t *TestReplaceMapper) Channel(path string) (uuid.UUID, bool) {
	v, ok := t.ChannelMap[path]
	return v, ok
}

func (t *TestReplaceMapper) Group(name string) (uuid.UUID, bool) {
	v, ok := t.GroupMap[name]
	return v, ok
}

func (t *TestReplaceMapper) User(name string) (uuid.UUID, bool) {
	v, ok := t.UserMap[name]
	return v, ok
}

func TestReplacer_Replace(t *testing.T) {
	t.Parallel()

	re := NewReplacer(&TestReplaceMapper{
		ChannelMap: map[string]uuid.UUID{
			"a": uuid.Must(uuid.FromString("ea452867-553b-4808-a14f-a47ee0009ee6")),
		},
		UserMap: map[string]uuid.UUID{
			"takashi_trap": uuid.Must(uuid.FromString("dfdff0c9-5de0-46ee-9721-2525e8bb3d45")),
			"takashi_trape": uuid.Must(uuid.FromString("dfdff0c9-5de0-46ee-9721-2525e8bb3d46")),
			"very_long_long_long_long_lo_name": uuid.Must(uuid.FromString("dfdff0c9-5de0-46ee-9721-2525e8bb3d47")),
		},
		GroupMap: map[string]uuid.UUID{
			"okあok": uuid.Must(uuid.FromString("dfabf0c9-5de0-46ee-9721-2525e8bb3d45")),
			"takashi_trapo": uuid.Must(uuid.FromString("dfabf0c9-5de0-46ee-9721-2525e8bb3d46")),
		},
	})

	tt := [][]string{
		{
			"aaaa#aeee `#a` @takashi_trapa @takashi_trap @#a\n```\n#a @takashi_trap\n```\n@okあok",
			"aaaa#aeee `#a` @takashi_trapa !{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"} @!{\"type\":\"channel\",\"raw\":\"#a\",\"id\":\"ea452867-553b-4808-a14f-a47ee0009ee6\"}\n```\n#a @takashi_trap\n```\n!{\"type\":\"group\",\"raw\":\"@okあok\",\"id\":\"dfabf0c9-5de0-46ee-9721-2525e8bb3d45\"}",
		},
		{
			"$$\\text{@takashi_trap}$$",
			"$$\\text{@takashi_trap}$$",
		},
		{
			"$$\n```\n@takashi_trap\n```\n$$",
			"$$\n```\n@takashi_trap\n```\n$$",
		},
		{
			"`$@takashi_trap$` @takashi_trap @very_long_long_long_long_lo_name",
			"`$@takashi_trap$` !{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"} !{\"type\":\"user\",\"raw\":\"@very_long_long_long_long_lo_name\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d47\"}",
		},
		{
			"`@takashi_trap` $@takashi_trap$ $$ $ `$@takashi_trap$$@takashi_trap$`$@takashi_trap$`$`",
			"`@takashi_trap` $@takashi_trap$ $$ $ `$@takashi_trap$$@takashi_trap$`$@takashi_trap$`$`",
		},
		{
			"`$@takashi_trap$` $@takashi_trap$ `@takashi_trap`",
			"`$@takashi_trap$` $@takashi_trap$ `@takashi_trap`",
		},
		{
			"`okあok`",
			"`okあok`",
		},
		{
			"$okあok$",
			"$okあok$",
		},
		{
			"`$okあok$`",
			"`$okあok$`",
		},
		{
			"````\n```\n@takashi_trap\n```\n````\n\n```\n@takashi_trap\n```",
			"````\n```\n@takashi_trap\n```\n````\n\n```\n@takashi_trap\n```",
		},
		{
			"@takashi_trapああ a@takashi_trap",
			"!{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"}ああ a!{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"}",
		},
		{
			":@takashi_trap:ああ a@takashi_trap",
			":@takashi_trap:ああ a!{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"}",
		},
		{
			"@takashi_trapああ:",
			"!{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"}ああ:",
		},
		{
			"@takashi_trap:@takashi_trap: :@takashi_trap: :takashi_trap",
			"!{\"type\":\"user\",\"raw\":\"@takashi_trap\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d45\"}:@takashi_trap: :@takashi_trap: :takashi_trap",
		},
		{
			"@takashi_trapo @takashi_trape",
			"!{\"type\":\"group\",\"raw\":\"@takashi_trapo\",\"id\":\"dfabf0c9-5de0-46ee-9721-2525e8bb3d46\"} !{\"type\":\"user\",\"raw\":\"@takashi_trape\",\"id\":\"dfdff0c9-5de0-46ee-9721-2525e8bb3d46\"}",
		},
	}
	for _, v := range tt {
		assert.Equal(t, v[1], re.Replace(v[0]))
	}
}
