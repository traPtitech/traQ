package jsonutil

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pointerMarshaler map[string]struct{}

func (*pointerMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`"called"`), nil
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	var (
		zero = struct {
			A int      `json:"a,omitempty"`
			B bool     `json:"b,omitempty"`
			C string   `json:"c,omitempty"`
			D *int     `json:"d,omitempty"`
			E []string `json:"e,omitempty"`
		}{}
		embedded = struct {
			struct1
			B int `json:"b"`
		}{struct1{A: 1}, 2}
	)

	tests := []struct {
		name string
		v    any
		want string
	}{
		{"nil slice is null", []string(nil), `null`},
		{"nil map is null", map[string]int(nil), `null`},
		{"empty slice", []string{}, `[]`},
		{"html is not escaped", "<a&b>", `"<a&b>"`},
		{"line separator is not escaped", " ", "\" \""},
		{"invalid utf8 is replaced", "a\xffb", "\"a�b\""},
		{"omitempty omits zero values", zero, `{}`},
		{"embedded fields are inlined", embedded, `{"a":1,"b":2}`},
		{"map keys are sorted", map[string]int{"b": 2, "a": 1, "c": 3}, `{"a":1,"b":2,"c":3}`},
		{"bytes are base64", []byte("hi"), `"aGk="`},
		{"duration is nanoseconds", time.Second, `1000000000`},
		{
			"pointer receiver method is called on non addressable value",
			struct {
				M pointerMarshaler `json:"m"`
			}{pointerMarshaler{"x": {}}},
			`{"m":"called"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Marshal(tt.v)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

type struct1 struct {
	A int `json:"a"`
}

func TestMarshalIndent(t *testing.T) {
	t.Parallel()

	got, err := MarshalIndent(map[string]int{"a": 1}, "  ")
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"a\": 1\n}", string(got))
}

func TestMarshalToString(t *testing.T) {
	t.Parallel()

	got, err := MarshalToString(map[string]int{"a": 1})
	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, got)
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	type target struct {
		Name string `json:"name"`
	}

	t.Run("names are case insensitive", func(t *testing.T) {
		t.Parallel()
		var v target
		require.NoError(t, Unmarshal([]byte(`{"NAME":"a"}`), &v))
		assert.Equal(t, "a", v.Name)
	})

	t.Run("duplicate names are allowed", func(t *testing.T) {
		t.Parallel()
		var v target
		require.NoError(t, Unmarshal([]byte(`{"name":"a","name":"b"}`), &v))
		assert.Equal(t, "b", v.Name)
	})

	t.Run("invalid utf8 is allowed", func(t *testing.T) {
		t.Parallel()
		var v target
		require.NoError(t, Unmarshal([]byte("{\"name\":\"a\xffb\"}"), &v))
		assert.Equal(t, "a�b", v.Name)
	})

	t.Run("trailing data is an error", func(t *testing.T) {
		t.Parallel()
		var v target
		assert.Error(t, Unmarshal([]byte(`{"name":"a"} x`), &v))
	})
}

func TestEncode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, Encode(&buf, map[string]int{"a": 1}))
	assert.Equal(t, "{\"a\":1}\n", buf.String())
}

func TestDecode(t *testing.T) {
	t.Parallel()

	type target struct {
		Name string `json:"name"`
	}

	t.Run("trailing data is ignored", func(t *testing.T) {
		t.Parallel()
		var v target
		require.NoError(t, Decode(bytes.NewBufferString(`{"NAME":"a"} x`), &v))
		assert.Equal(t, "a", v.Name)
	})

	t.Run("empty input is an error", func(t *testing.T) {
		t.Parallel()
		var v target
		assert.Error(t, Decode(bytes.NewBufferString(``), &v))
	})
}
