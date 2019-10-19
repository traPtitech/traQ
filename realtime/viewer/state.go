package viewer

import "strings"

// State 閲覧状態
type State int

const (
	// StateNone バックグランド表示中
	StateNone State = iota
	// StateMonitoring メッセージ表示中
	StateMonitoring
	// StateEditing メッセージ入力中
	StateEditing
)

// String string表記にします
func (s State) String() string {
	return viewStateStrings[s]
}

// FromString stringからviewer.Stateに変換します
func FromString(s string) State {
	return stringViewStates[strings.ToLower(s)]
}

var viewStateStrings = map[State]string{
	StateNone:       "none",
	StateEditing:    "editing",
	StateMonitoring: "monitoring",
}

var stringViewStates map[string]State

func init() {
	stringViewStates = map[string]State{}
	for v, k := range viewStateStrings {
		stringViewStates[k] = v
	}
}
