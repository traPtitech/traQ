# 通知仕様
イベント名毎に仕様を記載する。

SSEはServerSentEventsで通知されるdataの中身を表す。

FCMはFirebaseCloudMessagingで通知される情報を表す。記載されていない場合は、そのイベントはFCMで通知されない。


## USER_JOINED
ユーザーが新規登録された。

### SSE
TODO

## USER_LEFT
ユーザーが脱退した。

### SSE
TODO

## USER_TAGS_UPDATED
ユーザーのタグが更新された。

### SSE
+ `id`: タグが更新されたユーザーのId

## CHANNEL_CREATED
チャンネルが新規作成された。

### SSE
+ `id`: 作成されたチャンネルのId

## CHANNEL_DELETED
チャンネルが削除された。

### SSE
+ `id`: 削除されたチャンネルのId

## CHANNEL_UPDATED
チャンネルの名前またはトピックが変更された。

### SSE
+ `id`: 変更があったチャンネルのId

## CHANNEL_STARED
自分がチャンネルをスターした。
端末間同期目的に使用される。

### SSE
+ `id`: スターしたチャンネルのId

## CHANNEL_UNSTARED
自分がチャンネルのスターを解除した。
端末間同期目的に使用される。

### SSE
+ `id`: スターしたチャンネルのId

## CHANNEL_VISIBILITY_CHANGED
チャンネルの可視状態が変更された。

### SSE
TODO

## MESSAGE_CREATED
メッセージが投稿された。

### SSE
+ `id`: 投稿されたメッセージのId

### FCM
TODO

## MESSAGE_UPDATED
メッセージが更新された。

### SSE
+ `id`: 更新されたメッセージのId

### FCM
TODO

## MESSAGE_DELETED
メッセージが削除された。

### SSE
+ `id`: 削除されたメッセージのId

## MESSAGE_READ
自分がメッセージを読んだ。
端末間同期目的に使用される。

### SSE
TODO

## MESSAGE_STAMPED
メッセージにスタンプが押された。

### SSE
TODO

## MESSAGE_UNSTAMPED
メッセージからスタンプが外された。

### SSE
TODO

## MESSAGE_PINNED
メッセージがピン留めされた。

### SSE
TODO

## MESSAGE_UNPINNED
ピン留めされたメッセージのピンが外された。

### SSE
TODO

## MESSAGE_CLIPPED
自分がメッセージをクリップした。
端末間同期目的に使用される。

### SSE
+ `id`: クリップしたメッセージのId

## MESSAGE_UNCLIPPED
自分がメッセージをアンクリップした。
端末間同期目的に使用される。

### SSE
+ `id`: アンクリップしたメッセージのId

## STAMP_CREATED
スタンプが新しく追加された。

### SSE
TODO

## STAMP_DELETED
スタンプが削除された。

### SSE
TODO

## TRAQ_UPDATED
traQが更新された。

### SSE
TODO