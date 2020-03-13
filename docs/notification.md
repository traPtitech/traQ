# 通知仕様
イベント名毎に仕様を記載する。

SSEはServerSentEventsで通知されるdataの中身を表す。

FCMはFirebaseCloudMessagingで通知される情報を表す。記載されていない場合は、そのイベントはFCMで通知されない。


## USER_JOINED
ユーザーが新規登録された。

### SSE
対象: 全員

+ `id`: 登録されたユーザーのId

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

## USER_GROUP_CREATED
ユーザーグループが作成された

### SSE
対象: 全員

+ `id`: 作成されたユーザーグループのId

## USER_GROUP_UPDATED
ユーザーグループが更新された

### SSE
対象: 全員

+ `id`: 作成されたユーザーグループのId

## USER_GROUP_DELETED
ユーザーグループが削除された

### SSE
対象: 全員

+ `id`: 削除されたユーザーグループのId

## USER_ICON_UPDATED
ユーザーのアイコンが更新された。

### SSE
対象: 全員

+ `id`: アイコンが更新されたユーザーのId

## USER_ONLINE
ユーザーがオンラインになった。

### SSE
対象: 全員

+ `id`: オンラインになったユーザーのId

## USER_OFFLINE
ユーザーがオフラインになった。

### SSE
対象: 全員

+ `id`: オフラインになったユーザーのId

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

## MESSAGE_CREATED
メッセージが投稿された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー・投稿チャンネルに通知をつけているユーザー・メンションを受けたユーザー

+ `id`: 投稿されたメッセージのId

### FCM
#### data
+ `title`: チャンネル名
+ `body`: ユーザー名+メッセージ本体(100文字まで)
+ `path`: チャンネルのパス(`/users/:userID`または`/channels/:channelPath`)
+ `icon`: ユーザーアイコン
+ `tag`: `c:(チャンネルID)`
+ `image`: 添付ファイルに画像があればその(１つ目の)サムネイル画像のURL

## MESSAGE_UPDATED
メッセージが更新された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 更新されたメッセージのId

## MESSAGE_DELETED
メッセージが削除された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `id`: 削除されたメッセージのId

## MESSAGE_READ
自分があるチャンネルのメッセージを読んだ。
端末間同期目的に使用される。

### SSE
対象: イベント発生元ユーザー

+ `id`: 読んだチャンネルId

## MESSAGE_STAMPED
メッセージにスタンプが押された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `message_id`: メッセージId
+ `user_id`: スタンプを押したユーザーのId
+ `stamp_id`: スタンプのId
+ `count`: そのユーザーが押した数
+ `created_at`: そのユーザーがそのスタンプをそのメッセージに最初に押した日時

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

+ `message_id`: ピンされたメッセージのID
+ `channel_id`: ピンされたメッセージのチャンネルID

## MESSAGE_UNPINNED
ピン留めされたメッセージのピンが外された。

### SSE
対象: 投稿チャンネルにハートビートを送信しているユーザー

+ `message_id`: ピンが外されたメッセージのID
+ `channel_id`: ピンが外されたメッセージのチャンネルID

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

## USER_WEBRTC_STATE_CHANGED
ユーザーのWebRTCの状態が変化した

### SSE
対象: 全員

+ `user_id`: 変更があったユーザーのId
+ `channel_id`: ユーザーの変更後の接続チャンネルのId
+ `state`: ユーザーの変更後の状態(配列)
