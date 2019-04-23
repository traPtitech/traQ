package main

import (
	"context"
	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"go.uber.org/zap"
	"golang.org/x/exp/utf8string"
	"google.golang.org/api/option"
)

// FCMManager Firebaseマネージャー構造体
type FCMManager struct {
	messaging *messaging.Client
	repo      repository.Repository
	hub       *hub.Hub
	logger    *zap.Logger
	origin    string
}

// NewFCMManager FCMManagerを生成します
func NewFCMManager(repo repository.Repository, hub *hub.Hub, logger *zap.Logger, serviceAccountFile, origin string) (*FCMManager, error) {
	manager := &FCMManager{
		repo:   repo,
		hub:    hub,
		logger: logger,
		origin: origin,
	}

	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsFile(serviceAccountFile))
	if err != nil {
		return nil, err
	}

	manager.messaging, err = app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}

	go func() {
		sub := hub.Subscribe(100, event.MessageCreated)
		for ev := range sub.Receiver {
			m := ev.Fields["message"].(*model.Message)
			p := ev.Fields["plain"].(string)
			e := ev.Fields["embedded"].([]*message.EmbeddedInfo)
			go manager.processMessageCreated(m, p, e)
		}
	}()
	return manager, nil
}

func (m *FCMManager) processMessageCreated(message *model.Message, plain string, embedded []*message.EmbeddedInfo) {
	logger := m.logger.With(zap.Stringer("messageId", message.ID))

	// チャンネル情報を取得
	ch, err := m.repo.GetChannel(message.ChannelID)
	if err != nil {
		logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
		return
	}

	// 投稿ユーザー情報を取得
	mUser, err := m.repo.GetUser(message.UserID)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", message.UserID)) // 失敗
		return
	}

	// データ初期化
	data := map[string]string{
		"title":     "traQ",
		"icon":      fmt.Sprintf("%s/api/1.0/public/icon/%s", m.origin, mUser.Icon),
		"vibration": "[1000, 1000, 1000]",
		"tag":       fmt.Sprintf("c:%s", message.ChannelID),
		"badge":     fmt.Sprintf("%s/static/badge.png", m.origin),
	}

	// メッセージボディ作成
	body := ""
	if ch.IsDMChannel() {
		if len(mUser.DisplayName) == 0 {
			data["title"] = "@" + mUser.Name
		} else {
			data["title"] = "@" + mUser.DisplayName
		}
		data["path"] = "/users/" + mUser.Name
		body = plain
	} else {
		path, err := m.repo.GetChannelPath(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("channelId", message.ChannelID))
			return
		}

		data["title"] = "#" + path
		data["path"] = "/channels/" + path

		if len(mUser.DisplayName) == 0 {
			body = fmt.Sprintf("%s: %s", mUser.Name, plain)
		} else {
			body = fmt.Sprintf("%s: %s", mUser.DisplayName, plain)
		}
	}

	if s := utf8string.NewString(body); s.RuneCount() > 100 {
		body = s.Slice(0, 97) + "..."
	}
	data["body"] = body

	for _, v := range embedded {
		if v.Type == "file" {
			if f, _ := m.repo.GetFileMeta(uuid.FromStringOrNil(v.ID)); f != nil && f.HasThumbnail {
				data["image"] = fmt.Sprintf("%s/api/1.0/files/%s/thumbnail", m.origin, v.ID)
				break
			}
		}
	}

	// 対象者計算
	targets := map[uuid.UUID]bool{}
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, err := m.repo.GetUsers()
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		for _, v := range users {
			if v.Bot {
				continue
			}
			targets[v.ID] = true
		}

	case !ch.IsPublic: // プライベートチャンネル
		pUsers, err := m.repo.GetPrivateChannelMemberIDs(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		addIDsToSet(targets, pUsers)

	default: // 通常チャンネルメッセージ
		users, err := m.repo.GetSubscribingUserIDs(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetSubscribingUserIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		addIDsToSet(targets, users)

		// ユーザーグループ・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err == nil {
					addIDsToSet(targets, []uuid.UUID{uid})
				}
			case "group":
				gs, err := m.repo.GetUserGroupMemberIDs(uuid.FromStringOrNil(v.ID))
				if err != nil {
					logger.Error("failed to GetUserGroupMemberIDs", zap.Error(err), zap.String("groupId", v.ID)) // 失敗
					return
				}
				addIDsToSet(targets, gs)
			}
		}

		// ミュート除外
		muted, err := m.repo.GetMuteUserIDs(message.ChannelID)
		if err != nil {
			logger.Error("failed to GetMuteUserIDs", zap.Error(err), zap.Stringer("channelId", message.ChannelID)) // 失敗
			return
		}
		deleteIDsFromSet(targets, muted)
	}
	delete(targets, message.UserID) // 自分を除外

	// 送信
	for u := range targets {
		go func(u uuid.UUID) {
			devs, err := m.repo.GetDeviceTokensByUserID(u)
			if err != nil {
				logger.Error("failed to GetDeviceTokensByUserID", zap.Error(err), zap.Stringer("userId", u)) // 失敗
				return
			}

			payload := &messaging.Message{
				Data: data,
				Android: &messaging.AndroidConfig{
					Priority: "high",
				},
				APNS: &messaging.APNSConfig{
					Payload: &messaging.APNSPayload{
						Aps: &messaging.Aps{
							Alert: &messaging.ApsAlert{
								Title: data["title"],
								Body:  data["body"],
							},
							Sound:    "default",
							ThreadID: data["tag"],
						},
					},
				},
			}
			for _, token := range devs {
				payload.Token = token
				err := backoff.Retry(func() error {
					if _, err := m.messaging.Send(context.Background(), payload); err != nil {
						switch {
						case messaging.IsRegistrationTokenNotRegistered(err):
							if err := m.repo.UnregisterDevice(token); err != nil {
								return backoff.Permanent(err)
							}
						case messaging.IsInvalidArgument(err):
							return backoff.Permanent(err)
						case messaging.IsServerUnavailable(err):
							fallthrough
						case messaging.IsInternal(err):
							fallthrough
						case messaging.IsMessageRateExceeded(err):
							fallthrough
						case messaging.IsUnknown(err):
							return err
						default:
							return err
						}
					}
					return nil
				}, backoff.NewExponentialBackOff())
				if err != nil {
					logger.Error("an error occurred in sending fcm", zap.Error(err), zap.String("deviceToken", token))
				}
			}
		}(u)
	}
}

func addIDsToSet(set map[uuid.UUID]bool, ids []uuid.UUID) {
	for _, v := range ids {
		set[v] = true
	}
}

func deleteIDsFromSet(set map[uuid.UUID]bool, ids []uuid.UUID) {
	for _, v := range ids {
		delete(set, v)
	}
}
