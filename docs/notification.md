# 通知仕様
イベント名毎に仕様を記載する。

SSEはServerSentEventsで通知されるdataの中身を表す。

FCMはFirebaseCloudMessagingで通知される情報を表す。記載されていない場合は、そのイベントはFCMで通知されない。


## USER_JOINED
ユーザーが新規登録された。

### SSE
対象: 全員

TODO

## USER_LEFT
ユーザーが脱退した。

### SSE
対象: 全員

TODO

## USER_UPDATED
ユーザーの情報が更新された。

### SSE
対象: 全員

+ `id`: 情報が更新されたユーザーのId

## USER_TAGS_UPDATED
ユーザーのタグが更新された。

### SSE
対象: 全員

+ `id`: タグが更新されたユーザーのId

## USER_ICON_UPDATED
ユーザーのアイコンが更新された。

### SSE
対象: 全員

+ `id`: アイコンが更新されたユーザーのId

## CHANNEL_CREATED
チャンネルが新規作成された。

### SSE
対象: 全員

+ `id`: 作成されたチャンネルのId

## CHANNEL_DELETED
チャンネルが削除された。

### SSE
対象: 全員

+ `id`: 削除されたチャンネルのId

## CHANNEL_UPDATED
チャンネルの名前またはトピックが変更された。

### SSE
対象: 全員

+ `id`: 変更があったチャンネルのId

## CHANNEL_STARED
自分がチャンネルをスターした。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `id`: スターしたチャンネルのId

## CHANNEL_UNSTARED
自分がチャンネルのスターを解除した。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `id`: スターしたチャンネルのId

## CHANNEL_VISIBILITY_CHANGED
チャンネルの可視状態が変更された。

### SSE
対象: 全員

TODO

## MESSAGE_CREATED
メッセージが投稿された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー・投稿チャンネルに通知をつけているユーザー・メンションを受けたユーザー

+ `id`: 投稿されたメッセージのId

### FCM
TODO

## MESSAGE_UPDATED
メッセージが更新された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 更新されたメッセージのId

### FCM
TODO

## MESSAGE_DELETED
メッセージが削除された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 削除されたメッセージのId

## MESSAGE_READ
自分がメッセージを読んだ。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `ids`: 読んだメッセージのIdの配列

## MESSAGE_STAMPED
メッセージにスタンプが押された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `message_id`: メッセージId
+ `user_id`: スタンプを押したユーザーのId
+ `stamp_id`: スタンプのId
+ `count`: そのユーザーが押した数

## MESSAGE_UNSTAMPED
メッセージからスタンプが外された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `message_id`: メッセージId
+ `user_id`: スタンプを押したユーザーのId
+ `stamp_id`: スタンプのId

## MESSAGE_PINNED
メッセージがピン留めされた。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 作成されたピンID

## MESSAGE_UNPINNED
ピン留めされたメッセージのピンが外された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 外されたピンID

## MESSAGE_CLIPPED
自分がメッセージをクリップした。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `id`: クリップしたメッセージのId

## MESSAGE_UNCLIPPED
自分がメッセージをアンクリップした。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `id`: アンクリップしたメッセージのId

## STAMP_CREATED
スタンプが新しく追加された。

### SSE
対象: 全員

+ `id`: 作成されたスタンプのId

## STAMP_MODIFIED
スタンプが修正された。

### SSE
対象: 全員

+ `id`: 修正されたスタンプのId

## STAMP_DELETED
スタンプが削除された。

### SSE
対象: 全員

+ `id`: 削除されたスタンプのId

## TRAQ_UPDATED
traQ-UIが更新された。

### SSE
対象: 全員

TODO