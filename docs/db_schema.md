# dbスキーマ

## users
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ユーザーID |
| name | VARCHAR(32) | NOT NULL, UNIQUE | 英数字名 |
| display_name | VARCHAR(64) | NOT NULL | 表示名 |
| email | TEXT | NOT NULL | メールアドレス |
| password | CHAR(128) | NOT NULL | ハッシュ化されたパスワード |
| salt | CHAR(128) | NOT NULL | パスワードソルト |
| icon | CHAR(36) | NOT NULL | アイコンのファイルID |
| status | TINYINT | NOT NULL | アカウントの状態 |
| bot | BOOLEAN | NOT NULL | botアカウントか |
| role | TEXT | NOT NULL | ロール |
| twitter_id | VARCHAR(15) | NOT NULL | ツイッターID |
| last_online | TIMESTAMP(6) | | 最終オンライン日時 |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## rbac_overrides
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| permission | VARCHAR(50) | PRIMARY KEY | パーミッション名 |
| validity | BOOLEAN | NOT NULL | 有効かどうか |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |

## bots
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | botID |
| bot_user_id | CHAR(36) | NOT NULL UNIQUE | botユーザーID |
| description | TEXT | NOT NULL | 説明 |
| verificationToken | TEXT | NOT NULL | 確認コード |
| access_token_id | CHAR(36) | NOT NULL | アクセストークンのID |
| post_url | TEXT | NOT NULL | botのPOSTエンドポイント |
| subscribe_events | TEXT | NOT NULL | botの購読イベント |
| activated | BOOLEAN | NOT NULL | 活性化されているかどうか |
| install_code | VARCHAR(30) | NOT NULL UNIQUE | インストールコード |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## webhook_bots

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | webhookID |
| bot_user_id | CHAR(36) | NOT NULL UNIQUE | botユーザーID |
| description | TEXT | NOT NULL | 説明 |
| channel_id | CHAR(36) | NOT NULL | 投稿先のデフォルトチャンネルID |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## users_tags

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| tag_id | CHAR(36) | PRIMARY KEY | タグID |
| is_locked | BOOLEAN | NOT NULL | ロックされているか |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |

## tags

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY |タグID |
| name | VARCHAR(30) | NOT NULL UNIQUE | タグ文字列 |
| restricted | BOOLEAN | NOT NULL | 制限つきタグかどうか |
| type | VARCHAR(30) | NOT NULL | タグの種類 |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |

## channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | チャンネルID |
| name | VARCHAR(20) | NOT NULL | チャンネル名 |
| parent_id | CHAR(36) | NOT NULL | 親チャンネルのID |
| topic | TEXT | NOT NULL | チャンネルトピック |
| is_forced | BOOLEAN | NOT NULL | 強制通知チャンネルか | 
| is_public | BOOLEAN | NOT NULL | 公開チャンネルか |
| is_visible | BOOLEAN | NOT NULL | 表示チャンネルか |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| updater_id | CHAR(36) | NOT NULL | 更新したユーザーのID | 
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

parent_idとnameの複合ユニーク制約

## users_private_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| channel_id | CHAR(36) | PRIMARY KEY | (プライベート)チャンネルID |

## messages

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | メッセージID |
| user_id | CHAR(36) | NOT NULL | 投稿者のユーザーID |
| channel_id | CHAR(36) | NOT NULL | 投稿先のチャンネルID |
| text | TEXT | NOT NULL | 投稿内容 |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## messages_stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| message_id | CHAR(36) | PRIMARY KEY | メッセージID |
| stamp_id | CHAR(36) | PRIMARY KEY | スタンプID |
| user_id | CHAR(36) | PRIMARY KEY | スタンプを押したユーザーID |
| count | INT | NOT NULL | スタンプを押した回数 |
| created_at | TIMESTAMP(6) | NOT NULL | 押した日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |

## unreads

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| message_id | CHAR(36) | PRIMARY KEY | メッセージID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |

## devices

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| token | VARCHAR(190) | PRIMARY KEY | デバイストークン |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |

## stamps

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | スタンプID |
| name | VARCHAR(32) | NOT NULL, UNIQUE | スタンプ表示名 |
| creator_id | CHAR(36) | NOT NULL | 作成者のユーザーID |
| file_id | CHAR(36) | NOT NULL | スタンプのファイルID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## files

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ファイルID |
| name | TEXT | NOT NULL | ファイル名 |
| mime | TEXT | NOT NULL | MIMEタイプ |
| size | BIGINT | NOT NULL | ファイルサイズ |
| creator_id | CHAR(36) | NOT NULL | 投稿者のユーザーID |
| hash | CHAR(32) | NOT NULL | ハッシュ値 |
| manager | VARCHAR(30) | NOT NULL DEFAULT '' | マネージャー名(空文字はデフォルトマネージャー) |
| has_thumbnail | BOOLEAN | NOT NULL | サムネイルがあるか |
| thumbnail_width | INT | NOT NULL | サムネイルの幅 |
| thumbnail_height | INT | NOT NULL | サムネイルの高さ |
| created_at | TIMESTAMP(6) | NOT NULL | 投稿日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## stars

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| channel_id | CHAR(36) | PRIMARY KEY | チャンネルID |
| created_at | TIMESTAMP(6) | NOT NULL | スターした日時 |

## users_subscribe_channels

| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| user_id | CHAR(36) | PRIMARY KEY | ユーザーID |
| channel_id | CHAR(36) | PRIMARY KEY | 通知を受け取るチャンネルID |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |

## pins
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | ピン留めID |
| message_id | CHAR(36) | NOT NULL UNIQUE | メッセージID |
| user_id | CHAR(36) | NOT NULL | ピン留めしたユーザーのID |
| created_at | TIMESTAMP(6) | NOT NULL | ピン留めした日時 |

## oauth2_tokens
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | トークンID |
| client_id | CHAR(36) | NOT NULL | クライアントID |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| redirect_uri | TEXT | NOT NULL | リダイレクトURI |
| access_token | VARCHAR(36) | NOT NULL | アクセストークン |
| refresh_token | VARCHAR(36) | NOT NULL | リフレッシュトークン |
| scopes | TEXT | NOT NULL | 許可されたスコープ |
| expires_in | INT | NOT NULL | 有効秒数 |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |

## oauth2_clients
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| id | CHAR(36) | PRIMARY KEY | クライアントID |
| name | VARCHAR(32) | NOT NULL | クライアント名 |
| description | TEXT | NOT NULL | クライアント説明 |
| confidential | BOOLEAN | NOT NULL | コンフィデンシャルクライアントかどうか |
| creator_id | CHAR(36) | NOT NULL | 登録者ID |
| secret | VARCHAR(36) | NOT NULL | クライアントシークレット |
| redirect_uri | TEXT | NOT NULL | リダイレクトURI |
| scopes | TEXT | NOT NULL | 許可されたスコープ |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
| updated_at | TIMESTAMP(6) | NOT NULL | 更新日時 |
| deleted_at | TIMESTAMP(6) | | 削除日時 |

## oauth2_authorizes
| カラム名 | 型 | 属性 | 説明など | 
| --- | --- | --- | --- |
| code | VARCHAR(36) | PRIMARY KEY | 認可コード |
| client_id | CHAR(36) | NOT NULL | クライアントID |
| user_id | CHAR(36) | NOT NULL | ユーザーID |
| expires_in | INT | NOT NULL | 有効秒数 |
| redirect_uri | TEXT | NOT NULL | リダイレクトURI |
| scopes | TEXT | NOT NULL | 有効なスコープ |
| original_scopes | TEXT | NOT NULL | 要求されたスコープ |
| code_challenge | VARCHAR(128) | NOT NULL | PKCE Code Challenge |
| code_challenge_method | TEXT | NOT NULL | PKCE Code Challenge Method |
| nonce | TEXT | NOT NULL | Nonce |
| created_at | TIMESTAMP(6) | NOT NULL | 作成日時 |
