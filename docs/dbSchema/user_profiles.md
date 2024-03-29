# user_profiles

## Description

ユーザープロフィールテーブル

<details>
<summary><strong>Table Definition</strong></summary>

```sql
CREATE TABLE `user_profiles` (
  `user_id` char(36) NOT NULL,
  `bio` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `twitter_id` varchar(15) NOT NULL DEFAULT '',
  `last_online` datetime(6) DEFAULT NULL,
  `home_channel` char(36) DEFAULT NULL,
  `updated_at` datetime(6) DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  KEY `user_profiles_home_channel_channels_id_foreign` (`home_channel`),
  CONSTRAINT `user_profiles_home_channel_channels_id_foreign` FOREIGN KEY (`home_channel`) REFERENCES `channels` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `user_profiles_user_id_users_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

</details>

## Columns

| Name | Type | Default | Nullable | Children | Parents | Comment |
| ---- | ---- | ------- | -------- | -------- | ------- | ------- |
| user_id | char(36) |  | false |  | [users](users.md) | ユーザーUUID |
| bio | text |  | false |  |  | bio |
| twitter_id | varchar(15) | '' | false |  |  | Twitter ID |
| last_online | datetime(6) | NULL | true |  |  | 最終オンライン日時 |
| home_channel | char(36) | NULL | true |  | [channels](channels.md) | ホームチャンネルUUID |
| updated_at | datetime(6) | NULL | true |  |  | 更新日時 |

## Constraints

| Name | Type | Definition |
| ---- | ---- | ---------- |
| PRIMARY | PRIMARY KEY | PRIMARY KEY (user_id) |
| user_profiles_home_channel_channels_id_foreign | FOREIGN KEY | FOREIGN KEY (home_channel) REFERENCES channels (id) |
| user_profiles_user_id_users_id_foreign | FOREIGN KEY | FOREIGN KEY (user_id) REFERENCES users (id) |

## Indexes

| Name | Definition |
| ---- | ---------- |
| user_profiles_home_channel_channels_id_foreign | KEY user_profiles_home_channel_channels_id_foreign (home_channel) USING BTREE |
| PRIMARY | PRIMARY KEY (user_id) USING BTREE |

## Relations

```mermaid
erDiagram

"user_profiles" |o--|| "users" : "FOREIGN KEY (user_id) REFERENCES users (id)"
"user_profiles" }o--o| "channels" : "FOREIGN KEY (home_channel) REFERENCES channels (id)"

"user_profiles" {
  char_36_ user_id PK
  text bio
  varchar_15_ twitter_id
  datetime_6_ last_online
  char_36_ home_channel FK
  datetime_6_ updated_at
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
```

---

> Generated by [tbls](https://github.com/k1LoW/tbls)
