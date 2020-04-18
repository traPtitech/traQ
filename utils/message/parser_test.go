package message

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	u1 := uuid.Must(uuid.FromString("ee764d5f-71d9-4a40-bc7b-547d8d097c91"))
	u2 := uuid.Must(uuid.FromString("0f1f6d9e-fb5b-4209-8a6d-33a098e79691"))

	SetOrigin("http://localhost:3000")

	cases := map[string]ParseResult{
		"test message !{aaa": {
			PlainText: "test message !{aaa",
		},
		`{"test": "test"}!!{}`: {
			PlainText: `{"test": "test"}!!{}`,
		},
		`!{aiueo::::aaaaaaa}`: {
			PlainText: `!{aiueo::::aaaaaaa}`,
		},
		`test message !{"test": "test"}`: {
			PlainText: `test message !{"test": "test"}`,
		},
		`test message !{"raw": "@test","type":"user","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}`: {
			PlainText: `test message @test`,
			Mentions:  []uuid.UUID{u1},
		},
		`!{"raw": "","type":"file","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}test message !{"raw": "@test","type":"user","id":"0f1f6d9e-fb5b-4209-8a6d-33a098e79691"}!{"raw": "","type":"message","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}`: {
			PlainText:   `[添付ファイル]test message @test[引用メッセージ]`,
			Attachments: []uuid.UUID{u1},
			Mentions:    []uuid.UUID{u2},
			Citation:    []uuid.UUID{u1},
		},
		`!{ test message !{"raw": "@test","type":"user","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}`: {
			PlainText: `!{ test message @test`,
			Mentions:  []uuid.UUID{u1},
		},
		`!{ test message !{"raw": "@test","type":"group","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}`: {
			PlainText:     `!{ test message @test`,
			GroupMentions: []uuid.UUID{u1},
		},
		`!{ test message !{"raw": "#a/e","type":"channel","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"}`: {
			PlainText:   `!{ test message #a/e`,
			ChannelLink: []uuid.UUID{u1},
		},
		`!{ test message !{"raw": 1,"type":"user","id":"test_id"}`: {
			PlainText: `!{ test message !{"raw": 1,"type":"user","id":"test_id"}`,
		},
		`!{ test message !{"raw": "1","type":"user","id":"test_id"}`: {
			PlainText: `!{ test message !{"raw": "1","type":"user","id":"test_id"}`,
		},
		`!{ test message !{"raw": "1","type":"","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"} http://localhost:3000/files/ee764d5f-71d9-4a40-bc7b-547d8d097c91`: {
			PlainText:   `!{ test message !{"raw": "1","type":"","id":"ee764d5f-71d9-4a40-bc7b-547d8d097c91"} [添付ファイル]`,
			Attachments: []uuid.UUID{u1},
		},
		`http://localhost:3000/messages/0f1f6d9e-fb5b-4209-8a6d-33a098e79691 test message http://localhost:3000/files/ee764d5f-71d9-4a40-bc7b-547d8d097c91 http://localhost:3000/fiales/ee764d5f-71d9-4a40-bc7b-547d8d097c91http://localhost:3000/files/ee764d5fa-71d9-4a40-bc7b-547d8d097c91`: {
			PlainText:   `[引用メッセージ] test message [添付ファイル] http://localhost:3000/fiales/ee764d5f-71d9-4a40-bc7b-547d8d097c91http://localhost:3000/files/ee764d5fa-71d9-4a40-bc7b-547d8d097c91`,
			Attachments: []uuid.UUID{u1},
			Citation:    []uuid.UUID{u2},
		},
	}

	for m, exp := range cases {
		m := m
		exp := exp
		t.Run(m, func(t *testing.T) {
			t.Parallel()
			res := Parse(m)
			assert.EqualValues(t, exp, *res)
		})
	}
}
