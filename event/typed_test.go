package event

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name string
}

type testPayload struct {
	ID     uuid.UUID    `hub:"id"`
	Item   *testItem    `hub:"item"`
	Count  int          `hub:"count"`
	IDs    []uuid.UUID  `hub:"ids"`
	Reason fmt.Stringer `hub:"reason"`
}

func (testPayload) TopicName() string { return "test.typed" }

type testStringer string

func (s testStringer) String() string { return string(s) }

type otherTestPayload struct {
	ID uuid.UUID `hub:"other_id"`
}

func (otherTestPayload) TopicName() string { return "test.other" }

type emptyTestPayload struct{}

func (emptyTestPayload) TopicName() string { return "test.empty" }

func newTestPayload() (testPayload, hub.Fields) {
	id := uuid.Must(uuid.NewV7())
	item := &testItem{Name: "a"}
	ids := []uuid.UUID{uuid.Must(uuid.NewV7())}
	reason := testStringer("r")
	return testPayload{ID: id, Item: item, Count: 3, IDs: ids, Reason: reason},
		hub.Fields{"id": id, "item": item, "count": 3, "ids": ids, "reason": reason}
}

func TestFieldsOf(t *testing.T) {
	t.Parallel()

	t.Run("each field is stored under its tag key", func(t *testing.T) {
		t.Parallel()
		p, legacy := newTestPayload()
		f := FieldsOf(p)
		assert.Equal(t, legacy, f)
		// ポインタは同じ値 (同一オブジェクト) がそのまま格納される
		assert.Same(t, p.Item, f["item"])
	})
	t.Run("zero values are stored with their static types", func(t *testing.T) {
		t.Parallel()
		f := FieldsOf(testPayload{})
		assert.Equal(t, hub.Fields{
			"id":     uuid.Nil,
			"item":   (*testItem)(nil),
			"count":  0,
			"ids":    []uuid.UUID(nil),
			"reason": nil, // interface 型のフィールドが nil の場合は nil になる
		}, f)
	})
	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, hub.Fields{}, FieldsOf(emptyTestPayload{}))
	})
	t.Run("pointer payload panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { FieldsOf(&otherTestPayload{}) })
	})
}

type missingTagPayload struct {
	ID uuid.UUID
}

func (missingTagPayload) TopicName() string { return "test.invalid" }

type emptyTagPayload struct {
	ID uuid.UUID `hub:""`
}

func (emptyTagPayload) TopicName() string { return "test.invalid" }

type duplicatedKeyPayload struct {
	A uuid.UUID `hub:"id"`
	B uuid.UUID `hub:"id"`
}

func (duplicatedKeyPayload) TopicName() string { return "test.invalid" }

type unexportedFieldPayload struct {
	id uuid.UUID `hub:"id"`
}

func (unexportedFieldPayload) TopicName() string { return "test.invalid" }

func TestInvalidPayloadType(t *testing.T) {
	t.Parallel()
	_ = unexportedFieldPayload{}.id

	cases := map[string]struct {
		fieldsOf func()
		decode   func()
	}{
		"missing tag": {
			func() { FieldsOf(missingTagPayload{}) },
			func() { PayloadOf[missingTagPayload](hub.Fields{"id": uuid.Nil}) },
		},
		"empty tag": {
			func() { FieldsOf(emptyTagPayload{}) },
			func() { PayloadOf[emptyTagPayload](hub.Fields{"": uuid.Nil}) },
		},
		"duplicated key": {
			func() { FieldsOf(duplicatedKeyPayload{}) },
			func() { PayloadOf[duplicatedKeyPayload](hub.Fields{"id": uuid.Nil}) },
		},
		"unexported field": {
			func() { FieldsOf(unexportedFieldPayload{}) },
			func() { PayloadOf[unexportedFieldPayload](hub.Fields{"id": uuid.Nil}) },
		},
		"interface type argument": {
			nil, // FieldsOf は具体的な値を受け取るため該当しない
			func() { PayloadOf[Payload](hub.Fields{}) },
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if c.fieldsOf != nil {
				assert.Panics(t, c.fieldsOf)
			}
			assert.Panics(t, c.decode)
		})
	}
}

func TestMustPayload(t *testing.T) {
	t.Parallel()

	t.Run("decodes legacy fields", func(t *testing.T) {
		t.Parallel()
		want, legacy := newTestPayload()
		got := MustPayload[testPayload](legacy)
		assert.Equal(t, want, got)
		assert.Same(t, want.Item, got.Item)
	})
	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		want, _ := newTestPayload()
		assert.Equal(t, want, MustPayload[testPayload](FieldsOf(want)))
	})
	t.Run("extra keys are ignored", func(t *testing.T) {
		t.Parallel()
		want, legacy := newTestPayload()
		legacy["extra"] = "x"
		assert.Equal(t, want, MustPayload[testPayload](legacy))
	})
	t.Run("typed nil pointer is accepted", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		legacy["item"] = (*testItem)(nil)
		assert.Nil(t, MustPayload[testPayload](legacy).Item)
	})
	t.Run("missing key panics", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		delete(legacy, "count")
		assert.Panics(t, func() { MustPayload[testPayload](legacy) })
	})
	t.Run("type mismatch panics", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		legacy["count"] = int64(3)
		assert.Panics(t, func() { MustPayload[testPayload](legacy) })
	})
	t.Run("pointer to field type panics", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		legacy["item"] = testItem{Name: "a"}
		assert.Panics(t, func() { MustPayload[testPayload](legacy) })
	})
	t.Run("untyped nil panics", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		legacy["item"] = nil
		assert.Panics(t, func() { MustPayload[testPayload](legacy) })
	})
	t.Run("interface field accepts implementations", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		legacy["reason"] = testStringer("x")
		assert.Equal(t, "x", MustPayload[testPayload](legacy).Reason.String())

		legacy["reason"] = 1
		assert.Panics(t, func() { MustPayload[testPayload](legacy) })
	})
	t.Run("payload of another topic panics", func(t *testing.T) {
		t.Parallel()
		_, legacy := newTestPayload()
		assert.Panics(t, func() { MustPayload[otherTestPayload](legacy) })
	})
}

func TestPayloadOf(t *testing.T) {
	t.Parallel()
	want, legacy := newTestPayload()

	p, ok := PayloadOf[testPayload](legacy)
	require.True(t, ok)
	assert.Equal(t, want, p)

	_, ok = PayloadOf[otherTestPayload](legacy)
	assert.False(t, ok)

	broken := hub.Fields{}
	for k, v := range legacy {
		broken[k] = v
	}
	broken["id"] = want.ID.String()
	p, ok = PayloadOf[testPayload](broken)
	assert.False(t, ok)
	assert.Equal(t, testPayload{}, p)

	_, ok = PayloadOf[testPayload](nil)
	assert.False(t, ok)

	e, ok := PayloadOf[emptyTestPayload](nil)
	assert.True(t, ok)
	assert.Equal(t, emptyTestPayload{}, e)
}

func TestPublish(t *testing.T) {
	t.Parallel()
	h := hub.New()
	defer h.Close()
	sub := h.Subscribe(1, "test.typed")

	want, legacy := newTestPayload()
	Publish(h, want)

	msg := <-sub.Receiver
	assert.Equal(t, "test.typed", msg.Name)
	assert.Equal(t, legacy, msg.Fields)
	assert.Equal(t, want, MustPayload[testPayload](msg.Fields))
}

func TestPayloadFieldsConcurrent(t *testing.T) {
	t.Parallel()
	want, legacy := newTestPayload()
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Equal(t, legacy, FieldsOf(want))
			assert.Equal(t, want, MustPayload[testPayload](legacy))
		}()
	}
	wg.Wait()
}

// definedPayloads event package で定義している全てのペイロード型。
// Test で型が規約を満たしているかを確認するため、ペイロード型を追加したらここにも追加すること。
// interface 型のフィールドはゼロ値 (nil) だとデコードできないため、値を入れて登録すること。
var definedPayloads = []Payload{
	StampCreatedPayload{},
	StampUpdatedPayload{},
	StampDeletedPayload{},
	UserCreatedPayload{},
	UserUpdatedPayload{},
	UserIconUpdatedPayload{},
	UserActivatedPayload{},
	UserTagAddedPayload{},
	UserTagUpdatedPayload{},
	UserTagRemovedPayload{},
	UserGroupCreatedPayload{},
	UserGroupUpdatedPayload{},
	UserGroupDeletedPayload{},
	UserGroupMemberAddedPayload{},
	UserGroupMemberUpdatedPayload{},
	UserGroupMemberRemovedPayload{},
	UserGroupAdminAddedPayload{},
	UserGroupAdminRemovedPayload{},
}

// TestDefinedPayloads 定義済みのペイロード型が規約を満たし、FieldsOf で変換した値をデコードできることを確かめる
func TestDefinedPayloads(t *testing.T) {
	t.Parallel()
	for _, p := range definedPayloads {
		t.Run(reflect.TypeOf(p).String(), func(t *testing.T) {
			t.Parallel()
			var f hub.Fields
			require.NotPanics(t, func() { f = FieldsOf(p) })
			v := reflect.New(reflect.TypeOf(p)).Elem()
			require.NoError(t, decodeFields(v, f))
			assert.Equal(t, p, v.Interface())
		})
	}
}
