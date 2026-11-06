// Package event traQ 内部のイベントのトピック名とペイロード型を定義する。
//
// イベントは hub (github.com/leandro-lugaresi/hub) で配送する。
// hub 上ではペイロードを hub.Fields として運ぶが、発行側・受信側は
// Payload を実装した型を使い、hub.Fields を直接組み立てたり読んだりしない。
//
// # 発行
//
// ペイロード型の値を Publish に渡す。トピック名は Payload.TopicName で決まる。
//
//	event.Publish(h, event.StampCreatedPayload{StampID: s.ID, Stamp: s})
//
// hub.Hub.Publish の直接呼び出しは forbidigo で禁止している (event/typed.go とテストを除く)。
//
// # 受信
//
// hub.Subscribe で受け取った hub.Message の Fields を MustPayload で復元する。
// 1 つのハンドラで複数のトピックを受ける場合は hub.Message.Name で分岐する。
// 不正な値を無視したい場合は PayloadOf を使う。
//
//	p := event.MustPayload[event.StampCreatedPayload](ev.Fields)
//
// # 発行元
//
//   - 永続化に伴うイベントは、repository の実装がトランザクションの commit 後に発行する。
//   - service 内部の状態の変化を表すイベントは、その状態を持つ service が発行する。
//   - router からは発行しない。
//
// 既知の例外には発行箇所に NOTE コメントを付けている。
//
// # ペイロード型
//
//   - 名前は「トピック定数名 + Payload」とする (例: StampCreated → StampCreatedPayload)。
//   - struct の値型で定義し、値で発行する。全フィールドを exported にし、
//     `hub:"key"` タグで hub.Fields 上のキー名を指定する。
//   - ペイロード型とトピック名の定数 (topic.go) は、この package に置く。
//   - フィールドに発行元の package の型を使っていて、この package に置くと import が循環する
//     ペイロード型だけは、例外的に発行元の package の payload.go に置いている
//     (service/viewer, service/qall)。新しく例外を増やさないこと。
//
// # テスト
//
//   - この package に置いたペイロード型は typed_test.go の definedPayloads に登録する。
//   - 例外的にほかの package に置いたペイロード型は、その package の payload_test.go から
//     eventtest.AssertValidPayload を呼んで検証する。
package event
