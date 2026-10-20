# channels

## Description

チャンネルテーブル

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE `channels` (
  `id` char(36) NOT NULL,
  `name` varchar(20) NOT NULL,
  `parent_id` char(36) NOT NULL,
  `topic` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `is_forced` tinyint(1) NOT NULL DEFAULT 0,
  `is_public` tinyint(1) NOT NULL DEFAULT 0,
  `is_visible` tinyint(1) NOT NULL DEFAULT 0,
  `creator_id` char(36) NOT NULL,
  `updater_id` char(36) NOT NULL,
  `created_at` datetime(6) DEFAULT NULL,
  `updated_at` datetime(6) DEFAULT NULL,
  `deleted_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name_parent` (`name`,`parent_id`),
  KEY `idx_channel_channels_id_is_public_is_forced` (`id`,`is_public`,`is_forced`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

</details>

## Columns

| Name | Type | Default | Nullable | Children | Parents | Comment |
| ---- | ---- | ------- | -------- | -------- | ------- | ------- |
| id | char(36) |  | false | [channel_events](channel_events.md) [dm_channel_mappings](dm_channel_mappings.md) [files](files.md) [messages](messages.md) [stars](stars.md) [unreads](unreads.md) [users_private_channels](users_private_channels.md) [users_subscribe_channels](users_subscribe_channels.md) [user_profiles](user_profiles.md) [webhook_bots](webhook_bots.md) [channels](channels.md) |  | チャンネルUUID |
| name | varchar(20) |  | false |  |  | チャンネル名 |
| parent_id | char(36) |  | false |  | [channels](channels.md) | 親チャンネルUUID |
| topic | text |  | false |  |  | チャンネルトピック |
| is_forced | tinyint(1) | 0 | false |  |  | 強制通知チャンネルかどうか |
| is_public | tinyint(1) | 0 | false |  |  | 公開チャンネルかどうか |
| is_visible | tinyint(1) | 0 | false |  |  | 可視チャンネルかどうか |
| creator_id | char(36) |  | false |  | [users](users.md) | チャンネル作成者UUID |
| updater_id | char(36) |  | false |  | [users](users.md) | チャンネル更新者UUID |
| created_at | datetime(6) | NULL | true |  |  | チャンネル作成日時 |
| updated_at | datetime(6) | NULL | true |  |  | チャンネル更新日時 |
| deleted_at | datetime(6) | NULL | true |  |  | チャンネル削除日時 |

## Constraints

| Name | Type | Definition |
| ---- | ---- | ---------- |
| name_parent | UNIQUE | UNIQUE KEY name_parent (name, parent_id) |
| PRIMARY | PRIMARY KEY | PRIMARY KEY (id) |

## Indexes

| Name | Definition |
| ---- | ---------- |
| idx_channel_channels_id_is_public_is_forced | KEY idx_channel_channels_id_is_public_is_forced (id, is_public, is_forced) USING BTREE |
| PRIMARY | PRIMARY KEY (id) USING BTREE |
| name_parent | UNIQUE KEY name_parent (name, parent_id) USING BTREE |

## Relations

```mermaid
erDiagram

"channel_events" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"dm_channel_mappings" |o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"files" }o--o| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"messages" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"stars" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"unreads" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"users_private_channels" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"users_subscribe_channels" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"user_profiles" }o--o| "channels" : "FOREIGN KEY (home_channel) REFERENCES channels (id)"
"webhook_bots" }o--|| "channels" : "FOREIGN KEY (channel_id) REFERENCES channels (id)"
"channels" }o--|| "channels" : "Additional Relation"
"channels" }o--|| "users" : "Additional Relation"

"channels" {
  char_36_ id PK
  varchar_20_ name
  char_36_ parent_id
  text topic
  tinyint_1_ is_forced
  tinyint_1_ is_public
  tinyint_1_ is_visible
  char_36_ creator_id
  char_36_ updater_id
  datetime_6_ created_at
  datetime_6_ updated_at
  datetime_6_ deleted_at
}
"channel_events" {
  char_36_ event_id PK
  char_36_ channel_id FK
  varchar_30_ event_type
  text detail
  datetime_6_ date_time
}
"dm_channel_mappings" {
  char_36_ channel_id PK
  char_36_ user1 FK
  char_36_ user2 FK
}
"files" {
  char_36_ id PK
  text name
  text mime
  bigint_20_ size
  char_36_ creator_id FK
  char_32_ hash
  varchar_30_ type
  tinyint_1_ is_animated_image
  char_36_ channel_id FK
  datetime_6_ created_at
  datetime_6_ deleted_at
}
"messages" {
  char_36_ id PK
  char_36_ user_id FK
  char_36_ channel_id FK
  text text
  datetime_6_ created_at
  datetime_6_ updated_at
  datetime_6_ deleted_at
}
"stars" {
  char_36_ user_id PK
  char_36_ channel_id PK
}
"unreads" {
  char_36_ user_id PK
  char_36_ channel_id PK
  char_36_ message_id PK
  tinyint_1_ noticeable
  datetime_6_ message_created_at
}
"users_private_channels" {
  char_36_ user_id PK
  char_36_ channel_id PK
}
"users_subscribe_channels" {
  char_36_ user_id PK
  char_36_ channel_id PK
  tinyint_1_ mark
  tinyint_1_ notify
}
"user_profiles" {
  char_36_ user_id PK
  text bio
  varchar_15_ twitter_id
  datetime_6_ last_online
  char_36_ home_channel FK
  datetime_6_ updated_at
}
"webhook_bots" {
  char_36_ id PK
  char_36_ bot_user_id FK
  text description
  text secret
  char_36_ channel_id FK
  char_36_ creator_id FK
  datetime_6_ created_at
  datetime_6_ updated_at
  datetime_6_ deleted_at
}
"users" {
  char_36_ id PK
  varchar_32_ name
  varchar_32_ display_name
  char_128_ password
  char_128_ salt
  char_36_ icon
  tinyint_4_ status
  tinyint_1_ bot
  varchar_30_ role
  datetime_6_ created_at
  datetime_6_ updated_at
}
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
