# dbスキーマ
ID系は全部UUID(string)

## users
部員管理システムに全てあってもいいと思うけど、OSSだから単体でも動くように

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ユーザーID |
| name | VARCHAR(32) | NOT NULL, UNIQUE | 表示名 |
| email | TEXT | NOT NULL | メールアドレス |
| password | CHAR(128) | NOT NULL | ハッシュ化されたパスワード |
| salt | CHAR(128) | NOT NULL | パスワードソルト |
| icon | CHAR(36) | NOT NULL | アイコンのファイルID |
| status | TINYINT | NOT NULL | アカウントの状態 |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## users_authorities
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| authority_type | INT | NOT NULL | 権限の種類 |
| is_valid | BOOLEAN | NOT NULL | 権限の有無 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

user_idとauthority_typeの複合ユニーク制約

## users_tags

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | タグID |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| tag | TEXT | NOT NULL | タグ文字列 |
| is_locked | BOOLEAN | NOT NULL | ロックされているか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | チャンネルID |
| name | VARCHAR(20) | NOT NULL | チャンネル名 |
| parent_id | CHAR(36) | NOT NULL | 親チャンネルのID |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| topic | TEXT | | チャンネルトピック |
| is_forced | BOOLEAN | NOT NULL | 強制通知チャンネルか | 
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| is_public | BOOLEAN | NOT NULL | 公開チャンネルか |
| is_visible | BOOLEAN | NOT NULL | 表示チャンネルか |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
| updater_id | CHAR(36) | NOT NULL | 更新したユーザーのID | 
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

parent_idとnameの複合ユニーク制約

## users_private_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| channel_id | CHAR(36) | NOT NULL | (プライベート)チャンネルID |

user_idとchannel_idの複合主キー


## messages

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | メッセージID |
| user_id | CHAR(36) | NOT NULL | 投稿者のユーザーID |
| channel_id | CHAR(36) | | 投稿先のチャンネルID |
| text | TEXT | NOT NULL | 投稿内容 |
| is_shared | BOOLEAN | NOT NULL | 外部共有が許可されているか |
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| created_at | TIMESTAMP | NOT NULL | 投稿日時 |
| updater_id | CHAR(36)  | NOT NULL | 更新したユーザーのID |
| updated_at | TIMESTAMP | NOT NULL | 更新日時 |

## messages_stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| message_id | CHAR(36) | NOT NULL | メッセージID |
| stamp_id | CHAR(36) | NOT NULL | スタンプID |
| user_id | CHAR(36) | NOT NULL | スタンプを押したユーザーID |
| created_at | TIMESTAMP | NOT NULL | 押した日時 |

## unreads

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| message_id | CHAR(36) | NOT NULL | メッセージID |

user_idとmessage_idの複合主キー

## devices

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| type | ENUM('apn', 'fcm') | NOT NULL | トークンの種類 |
| token | TEXT | NOT NULL | デバイストークン |

user_id, type, tokenの複合主キー

## stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | スタンプID |
| name | VARCHAR(20) | NOT NULL, UNIQUE | スタンプ表示名 |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

## files

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ファイルID |
| name | TEXT | NOT NULL | ファイル名 |
| mime | TEXT | NOT NULL | MIMEタイプ |
| size | BIGINT | NOT NULL | ファイルサイズ |
| creator_id | CHAR(36) | NOT NULL | 投稿者のユーザーID |
| is_deleted | BOOLEAN | NOT NULL | 削除されているか |
| hash | CHAR(32) | NOT NULL | ハッシュ値 |
| created_at | TIMESTAMP | NOT NULL | 投稿日時 |

## stars

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| channel_id | CHAR(36) | NOT NULL | チャンネルID |
| created_at | TIMESTAMP | NOT NULL | スターした日時 |

user_idとchannel_idの複合主キー

## users_notified_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| channel_id | CHAR(36) | NOT NULL | 通知を受け取るチャンネルID |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

user_idとchannel_idの複合主キー

## users_invisible_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| channel_id | CHAR(36) | NOT NULL | 非表示にするチャンネルID |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |

user_idとchannel_idの複合主キー

## clips

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| message_id | CHAR(36) | NOT NULL | メッセージID |
| created_at | TIMESTAMP | NOT NULL | クリップした日時 |

## pins
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| channel_id | CHAR(36) | NOT NULL | チャンネルID |
| message_id | CHAR(36) | NOT NULL | メッセージID |
| user_id | CHAR(36) | NOT NULL | ピン留めしたユーザーのID |
| created_at | TIMESTAMP | NOT NULL | ピン留めした日時 |

channel_idとmessage_idの複合主キー

## tokens
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| access_token | TEXT | NOT NULL | OAuthAccessToken |
| access_token_secret | TEXT | NOT NULL | OAuthAccessTokenSecret |
| created_at | TIMESTAMP | NOT NULL | 作成日時 |
