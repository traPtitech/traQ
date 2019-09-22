package main

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/fcm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
	"go.uber.org/zap"
	"strings"
)

func processMessageCreated(c *fcm.Client, repo repository.Repository, logger *zap.Logger, origin string, message *model.Message, plain string, embedded []*message.EmbeddedInfo) {
	logger = logger.With(zap.Stringer("messageId", message.ID))

	// チャンネル情報を取得
	ch, err := repo.GetChannel(message.ChannelID)
	if err != nil {
		logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
		return
	}

	// 投稿ユーザー情報を取得
	mUser, err := repo.GetUser(message.UserID)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", message.UserID)) // 失敗
		return
	}
	if len(mUser.DisplayName) == 0 {
		mUser.DisplayName = mUser.Name
	}

	data := &fcm.Payload{
		Icon: fmt.Sprintf("%s/api/1.0/public/icon/%s", origin, strings.ReplaceAll(mUser.Name, "#", "%23")),
		Tag:  "c:" + message.ChannelID.String(),
	}

	// メッセージボディ作成
	if ch.IsDMChannel() {
		data.Title = "@" + mUser.DisplayName
		data.Path = "/users/" + mUser.Name
		data.SetBodyWithEllipsis(plain)
	} else {
		path, err := repo.GetChannelPath(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("channelId", message.ChannelID))
			return
		}
		data.Title = "#" + path
		data.Path = "/channels/" + path
		data.SetBodyWithEllipsis(mUser.DisplayName + ": " + plain)
	}

	for _, v := range embedded {
		if v.Type == "file" {
			if f, _ := repo.GetFileMeta(uuid.FromStringOrNil(v.ID)); f != nil && f.HasThumbnail {
				data.Image = fmt.Sprintf("%s/api/1.0/files/%s/thumbnail", origin, v.ID)
				break
			}
		}
	}

	// 対象者計算
	targets := set.UUIDSet{}
	q := repository.UsersQuery{}.Active().NotBot()
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, err := repo.GetUserIDs(q)
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		targets.Add(users...)

	case !ch.IsPublic: // プライベートチャンネル
		pUsers, err := repo.GetUserIDs(q.CMemberOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		targets.Add(pUsers...)

	default: // 通常チャンネルメッセージ
		users, err := repo.GetUserIDs(q.SubscriberOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetSubscribingUserIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		targets.Add(users...)

		// ユーザーグループ・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err == nil {
					// TODO 凍結ユーザーの除外
					// MEMO 凍結ユーザーはクライアント側で置換されないのでこのままでも問題はない
					targets.Add(uid)
				}
			case "group":
				gs, err := repo.GetUserIDs(q.GMemberOf(uuid.FromStringOrNil(v.ID)))
				if err != nil {
					logger.Error("failed to GetUserGroupMemberIDs", zap.Error(err), zap.String("groupId", v.ID)) // 失敗
					return
				}
				targets.Add(gs...)
			}
		}

		// ミュート除外
		muted, err := repo.GetMuteUserIDs(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetMuteUserIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		targets.Remove(muted...)
	}
	targets.Remove(message.UserID) // 自分を除外

	c.Send(targets, data)
}
