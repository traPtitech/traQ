# dbスキーマ
ID系は全部UUID(string)

## users
部員管理システムに全てあってもいいと思うけど、OSSだから単体でも動くように

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ユーザーID |
| name | VARCHAR(32) | NOT NULL, UNIQUE | 英数字名 |
| display_name | VARCHAR(32) | NOT NULL | 表示名 |
| email | TEXT | NOT NULL | メールアドレス |
| password | CHAR(128) | NOT NULL | ハッシュ化されたパスワード |
| salt | CHAR(128) | NOT NULL | パスワードソルト |
| icon | CHAR(36) | NOT NULL | アイコンのファイルID |
| status | TINYINT | NOT NULL | アカウントの状態 |
| bot | BOOLEAN | NOT NULL | botアカウントか |
| role | TEXT | NOT NULL | ロール |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## rbac_overrides
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | botユーザーID |
| permission | TEXT | NOT NULL | パーミッション名 |
| validity | BOOLEAN | NOT NULL | 有効かどうか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

user_idとpermissionの複合ユニーク制約

## bots
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY (外部キー) | botユーザーID |
| type | INT | NOT NULL | 1:汎用bot, 2:webhook |
| description | TEXT | NOT NULL | 説明 |
| is_valid | BOOLEAN | NOT NULL | 有効かどうか |
| creator_id | CHAR(36) | NOT NULL (外部キー) | 登録者 |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updater_id | CHAR(36) | NOT NULL (外部キー) | 更新者 | 
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## webhooks

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | NOT NULL PRIMARY KEY | webhookID |
| user_id | CHAR(36) | NOT NULL (外部キー) | botユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | 投稿先のデフォルトチャンネルID |

## users_tags

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY (外部キー) | ユーザーID |
| tag_id | CHAR(36) | NOT NULL (外部キー) | タグID |
| is_locked | BOOLEAN | NOT NULL | ロックされているか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## tags

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY |タグID |
| name | VARCHAR(30) | NOT NULL UNIQUE | タグ文字列 |

## channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | チャンネルID |
| name | VARCHAR(20) | NOT NULL | チャンネル名 |
| parent_id | CHAR(36) | NOT NULL | 親チャンネルのID |
| creator_id | CHAR(36) | NOT NULL (外部キー) | 作成者のユーザーID |
| topic | TEXT | | チャンネルトピック |
| is_forced | BOOLEAN | NOT NULL | 強制通知チャンネルか | 
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| is_public | BOOLEAN | NOT NULL | 公開チャンネルか |
| is_visible | BOOLEAN | NOT NULL | 表示チャンネルか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updater_id | CHAR(36) | NOT NULL (外部キー) | 更新したユーザーのID | 
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

parent_idとnameの複合ユニーク制約

## users_private_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | (プライベート)チャンネルID |

user_idとchannel_idの複合主キー

## messages

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | メッセージID |
| user_id | CHAR(36) | NOT NULL (外部キー) | 投稿者のユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | 投稿先のチャンネルID |
| text | TEXT | NOT NULL | 投稿内容 |
| is_shared | BOOLEAN | NOT NULL | 外部共有が許可されているか |
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| created_at | TIMESTAMP | NOT NULL | 投稿日時 |
| updater_id | CHAR(36)  | NOT NULL (外部キー) | 更新したユーザーのID |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## messages_stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| message_id | CHAR(36) | NOT NULL (外部キー) | メッセージID |
| stamp_id | CHAR(36) | NOT NULL (外部キー) | スタンプID |
| user_id | CHAR(36) | NOT NULL (外部キー) | スタンプを押したユーザーID |
| count | INT | NOT NULL | スタンプを押した回数 |
| created_at | TIMESTAMP | NOT NULL | 押した日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## unreads

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| message_id | CHAR(36) | NOT NULL (外部キー) | メッセージID |

user_idとmessage_idの複合主キー

## devices

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| token | VARCHAR(255) | NOT NULL PRIMARY KEY | デバイストークン |

## stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | スタンプID |
| name | VARCHAR(20) | NOT NULL, UNIQUE | スタンプ表示名 |
| creator_id | CHAR(36) | NOT NULL (外部キー) | 作成者のユーザーID |
| file_id | CHAR(36) | NOT NULL (外部キー) | スタンプのファイルID |
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## files

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ファイルID |
| name | TEXT | NOT NULL | ファイル名 |
| mime | TEXT | NOT NULL | MIMEタイプ |
| size | BIGINT | NOT NULL | ファイルサイズ |
| creator_id | CHAR(36) | NOT NULL (外部キー) | 投稿者のユーザーID |
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| hash | CHAR(32) | NOT NULL | ハッシュ値 |
| manager | VARCHAR(30) | NOT NULL DEFAULT '' | マネージャー名(空文字はデフォルトマネージャー) |
| has_thumbnail | BOOLEAN | NOT NULL | サムネイルがあるか |
| thumbnail_width | INT | NOT NULL | サムネイルの幅 |
| thumbnail_height | INT | NOT NULL | サムネイルの高さ |
| created_at | TIMESTAMP | NOT NULL | 投稿日時 |

## stars

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | チャンネルID |
| created_at | TIMESTAMP | NOT NULL | スターした日時 |

user_idとchannel_idの複合主キー

## users_subscribe_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | 通知を受け取るチャンネルID |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

user_idとchannel_idの複合主キー

## users_invisible_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | 非表示にするチャンネルID |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

user_idとchannel_idの複合主キー

## clips

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| message_id | CHAR(36) | NOT NULL (外部キー) | メッセージID |
| created_at | TIMESTAMP | NOT NULL | クリップした日時 |

## pins
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ピン留めID |
| channel_id | CHAR(36) | NOT NULL (外部キー) | チャンネルID |
| message_id | CHAR(36) | NOT NULL (外部キー) | メッセージID |
| user_id | CHAR(36) | NOT NULL (外部キー) | ピン留めしたユーザーのID |
| created_at | TIMESTAMP | NOT NULL | ピン留めした日時 |

## tokens
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL (外部キー) | ユーザーID |
| access_token | TEXT | NOT NULL | OAuthAccessToken |
| access_token_secret | TEXT | NOT NULL | OAuthAccessTokenSecret |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
