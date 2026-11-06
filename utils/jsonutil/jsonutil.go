// Package jsonutil traQのJSON形式を保ったままencoding/json/v2を利用するためのラッパーです。
package jsonutil

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
)

// options v1のセマンティクスに、json-iteratorのConfigFastestと同様にHTMLエスケープをしない設定を加えたもの
//
// MarshalJSON内で呼び出すMarshalにはオプションが引き継がれないため、traQ内のMarshalJSON実装でも本パッケージを利用してください。
var options = json.JoinOptions(
	jsonv1.DefaultOptionsV1(),
	jsonv1.CallMethodsWithLegacySemantics(false),
	jsontext.EscapeForHTML(false),
	jsontext.EscapeForJS(false),
)

// Marshal vをJSONにエンコードします。マップのキーはソートされます
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v, options)
}

// MarshalIndent vをインデント付きのJSONにエンコードします
func MarshalIndent(v any, indent string) ([]byte, error) {
	return json.Marshal(v, options, jsontext.WithIndent(indent))
}

// MarshalToString vをJSON文字列にエンコードします
func MarshalToString(v any) (string, error) {
	b, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Unmarshal dataをvにデコードします
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v, options)
}

// Encode vをエンコードし、末尾に改行を付けてwに書き込みます
func Encode(w io.Writer, v any) error {
	b, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// Decode rから最初のJSON値を読み込み、vにデコードします。以降のデータは読み捨てます
func Decode(r io.Reader, v any) error {
	return json.UnmarshalDecode(jsontext.NewDecoder(r, options), v, options)
}
