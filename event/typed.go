package event

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/leandro-lugaresi/hub"
)

// Payload は型付きイベントのペイロード
//
// `hub:"key"` タグで hub.Fields 上のキー名を指定して利用する
// 内部制約上、全ての property は exported でなければならない
//
// example:
//
//	type StampCreatedPayload struct {
//		StampID uuid.UUID    `hub:"stamp_id"`
//		Stamp   *model.Stamp `hub:"stamp"`
//	}
type Payload interface {
	TopicName() string
}

const fieldTag = "hub"

type payloadField struct {
	index int
	key   string
}

// payloadFieldsCache reflect.Type -> []payloadField
var payloadFieldsCache sync.Map

// payloadFields ペイロード型 t のフィールドとキー名の対応を返す。
// ペイロード型の定義が規約に反する場合は panic する
func payloadFields(t reflect.Type) []payloadField {
	if cached, ok := payloadFieldsCache.Load(t); ok {
		return cached.([]payloadField)
	}

	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("event: payload type %s must be a struct", t))
	}
	fields := make([]payloadField, 0, t.NumField())
	seen := make(map[string]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			panic(fmt.Sprintf("event: payload type %s: field %s must be exported", t, sf.Name))
		}
		key, ok := sf.Tag.Lookup(fieldTag)
		if !ok || key == "" {
			panic(fmt.Sprintf("event: payload type %s: field %s must have a %s tag", t, sf.Name, fieldTag))
		}
		if prev, dup := seen[key]; dup {
			panic(fmt.Sprintf("event: payload type %s: fields %s and %s have the same key %q", t, prev, sf.Name, key))
		}
		seen[key] = sf.Name
		fields = append(fields, payloadField{index: i, key: key})
	}

	actual, _ := payloadFieldsCache.LoadOrStore(t, fields)
	return actual.([]payloadField)
}

// FieldsOf は Payload の値を変換した hub.Fields を返す
func FieldsOf(p Payload) hub.Fields {
	v := reflect.ValueOf(p)
	fields := payloadFields(v.Type())
	f := make(hub.Fields, len(fields))
	for _, pf := range fields {
		f[pf.key] = v.Field(pf.index).Interface()
	}
	return f
}

// Publish は Payload を TopicName() で指定されるトピック名で hub に Publish する
func Publish(h *hub.Hub, p Payload) {
	h.Publish(hub.Message{Name: p.TopicName(), Fields: FieldsOf(p)})
}

// MustPayload hub.Fields から型 T のペイロードを組み立てる（失敗時に panic する）
func MustPayload[T Payload](f hub.Fields) T {
	p, err := decodePayload[T](f)
	if err != nil {
		panic(err)
	}
	return p
}

// PayloadOf hub.Fields から型 T のペイロードを組み立てる
func PayloadOf[T Payload](f hub.Fields) (T, bool) {
	p, err := decodePayload[T](f)
	if err != nil {
		var zero T
		return zero, false
	}
	return p, true
}

func decodePayload[T Payload](f hub.Fields) (T, error) {
	var p T
	err := decodeFields(reflect.ValueOf(&p).Elem(), f)
	return p, err
}

// decodeFields hub.Fields の値を struct の各フィールドに設定する
func decodeFields(v reflect.Value, f hub.Fields) error {
	t := v.Type()
	for _, pf := range payloadFields(t) {
		raw, ok := f[pf.key]
		if !ok {
			return fmt.Errorf("event: %s: key %q not found", t, pf.key)
		}
		fv := v.Field(pf.index)
		// x.(T) と同じ判定: nil は常に不一致、具体型は型が一致、interface 型は実装していれば一致
		if raw == nil {
			return fmt.Errorf("event: %s: key %q is nil, want %s", t, pf.key, fv.Type())
		}
		rv := reflect.ValueOf(raw)
		if fv.Kind() == reflect.Interface {
			if !rv.Type().Implements(fv.Type()) {
				return fmt.Errorf("event: %s: key %q is %s, want %s", t, pf.key, rv.Type(), fv.Type())
			}
		} else if rv.Type() != fv.Type() {
			return fmt.Errorf("event: %s: key %q is %s, want %s", t, pf.key, rv.Type(), fv.Type())
		}
		fv.Set(rv)
	}
	return nil
}
