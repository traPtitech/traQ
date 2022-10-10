package optional

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOf_ValueOrZero(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var o Of[int]
		assert.EqualValues(t, 0, o.ValueOrZero())
	})
	t.Run("invalid, has value", func(t *testing.T) {
		o := Of[int]{Valid: false, V: 123}
		assert.EqualValues(t, 0, o.ValueOrZero())
	})
	t.Run("valid", func(t *testing.T) {
		o := From(123)
		assert.EqualValues(t, 123, o.ValueOrZero())
	})
}

func TestOf_UnmarshalJSON(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalJSON([]byte("null"))
		if assert.NoError(t, err) {
			assert.False(t, o.Valid)
		}
	})
	t.Run("bool, true", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalJSON([]byte("true"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.True(t, o.V)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalJSON([]byte("false"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.False(t, o.V)
		}
	})
	t.Run("int", func(t *testing.T) {
		var o Of[int]
		err := o.UnmarshalJSON([]byte("123"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, 123, o.V)
		}
	})
	t.Run("string", func(t *testing.T) {
		var o Of[string]
		err := o.UnmarshalJSON([]byte("\"Hello\""))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "Hello", o.V)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		var o Of[time.Time]
		now, err := time.Parse(time.RFC3339, "2022-10-10T14:12:02Z")
		require.NoError(t, err)
		err = o.UnmarshalJSON([]byte("\"" + now.Format(time.RFC3339) + "\""))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, now, o.V)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		var o Of[uuid.UUID]
		err := o.UnmarshalJSON([]byte("\"b3b6173c-6dd4-45a6-bcb8-9b74acb037be\""))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "b3b6173c-6dd4-45a6-bcb8-9b74acb037be", o.V.String())
		}
	})
}

func TestOf_MarshalJSON(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var o Of[bool]
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "null", v)
		}
	})
	t.Run("bool, true", func(t *testing.T) {
		o := From(true)
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "true", v)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		o := From(false)
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "false", v)
		}
	})
	t.Run("int", func(t *testing.T) {
		o := From(123)
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "123", v)
		}
	})
	t.Run("string", func(t *testing.T) {
		o := From("World")
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "\"World\"", v)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		now := time.Now()
		o := From(now)
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "\""+now.Format(time.RFC3339Nano)+"\"", v)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4())
		o := From(id)
		v, err := o.MarshalJSON()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "\""+id.String()+"\"", v)
		}
	})
}

func TestOf_UnmarshalText(t *testing.T) {
	t.Run("invalid, empty", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalText([]byte{})
		if assert.NoError(t, err) {
			assert.False(t, o.Valid)
		}
	})
	t.Run("invalid, null", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalText([]byte("null"))
		if assert.NoError(t, err) {
			assert.False(t, o.Valid)
		}
	})
	t.Run("bool, true", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalText([]byte("true"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.True(t, o.V)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		var o Of[bool]
		err := o.UnmarshalText([]byte("false"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.False(t, o.V)
		}
	})
	t.Run("int", func(t *testing.T) {
		var o Of[int]
		err := o.UnmarshalText([]byte("123"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, 123, o.V)
		}
	})
	t.Run("string", func(t *testing.T) {
		var o Of[string]
		err := o.UnmarshalText([]byte("Hello"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "Hello", o.V)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		var o Of[time.Time]
		now, err := time.Parse(time.RFC3339, "2022-10-10T14:12:02Z")
		require.NoError(t, err)
		err = o.UnmarshalText([]byte("2022-10-10T14:12:02Z"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, now, o.V)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		var o Of[uuid.UUID]
		err := o.Scan([]byte("b3b6173c-6dd4-45a6-bcb8-9b74acb037be"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "b3b6173c-6dd4-45a6-bcb8-9b74acb037be", o.V.String())
		}
	})
}

func TestOf_Scan(t *testing.T) {
	t.Run("bool, true", func(t *testing.T) {
		var o Of[bool]
		err := o.Scan([]byte("true"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.True(t, o.V)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		var o Of[bool]
		err := o.Scan([]byte("false"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.False(t, o.V)
		}
	})
	t.Run("int", func(t *testing.T) {
		var o Of[int]
		err := o.Scan([]byte("123"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, 123, o.V)
		}
	})
	t.Run("string", func(t *testing.T) {
		var o Of[string]
		err := o.Scan([]byte("Hello"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "Hello", o.V)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		var o Of[time.Time]
		now := time.Now()
		err := o.Scan(now)
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, now, o.V)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		var o Of[uuid.UUID]
		err := o.Scan([]byte("b3b6173c-6dd4-45a6-bcb8-9b74acb037be"))
		if assert.NoError(t, err) {
			assert.True(t, o.Valid)
			assert.EqualValues(t, "b3b6173c-6dd4-45a6-bcb8-9b74acb037be", o.V.String())
		}
	})
}

func TestOf_MarshalText(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var o Of[bool]
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.Len(t, v, 0)
		}
	})
	t.Run("bool, true", func(t *testing.T) {
		o := From(true)
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "true", v)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		o := From(false)
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "false", v)
		}
	})
	t.Run("int", func(t *testing.T) {
		o := From(123)
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "123", v)
		}
	})
	t.Run("string", func(t *testing.T) {
		o := From("World")
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "World", v)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		now := time.Now()
		o := From(now)
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, now.Format(time.RFC3339Nano), v)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4())
		o := From(id)
		v, err := o.MarshalText()
		if assert.NoError(t, err) {
			assert.EqualValues(t, id.String(), v)
		}
	})
}

func TestOf_Value(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var o Of[bool]
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.Nil(t, v)
		}
	})
	t.Run("bool, true", func(t *testing.T) {
		o := From(true)
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, true, v)
		}
	})
	t.Run("bool, false", func(t *testing.T) {
		o := From(false)
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, false, v)
		}
	})
	t.Run("int", func(t *testing.T) {
		o := From(123)
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, 123, v)
			assert.IsType(t, int64(123), v)
		}
	})
	t.Run("string", func(t *testing.T) {
		o := From("World")
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, "World", v)
		}
	})
	t.Run("time.Time", func(t *testing.T) {
		now := time.Now()
		o := From(now)
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, now, v)
		}
	})
	t.Run("uuid.UUID", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4())
		o := From(id)
		v, err := o.Value()
		if assert.NoError(t, err) {
			assert.EqualValues(t, id.String(), v)
		}
	})
}
