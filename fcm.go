package main

import (
	"context"
	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"golang.org/x/exp/utf8string"
	"google.golang.org/api/option"
	"strings"
)

// FCMManager Firebaseマネージャー構造体
type FCMManager struct {
	messaging *messaging.Client
	repo      repository.Repository
	hub       *hub.Hub
	origin    string
}

// NewFCMManager FCMManagerを生成します
func NewFCMManager(repo repository.Repository, hub *hub.Hub, serviceAccountFile, origin string) (*FCMManager, error) {
	manager := &FCMManager{
		repo:   repo,
		hub:    hub,
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
	// make payload
	payload := map[string]string{
		"title":     "traQ",
		"icon":      fmt.Sprintf("%s/api/1.0/users/%s/icon?thumb", m.origin, message.UserID),
		"vibration": "[1000, 1000, 1000]",
		"tag":       fmt.Sprintf("c:%s", message.ChannelID),
		"badge":     fmt.Sprintf("%s/static/badge.png", m.origin),
	}
	ch, _ := m.repo.GetChannel(message.ChannelID)
	mUser, _ := m.repo.GetUser(message.UserID)
	body := ""
	if ch.IsDMChannel() {
		if mUser != nil {
			if len(mUser.DisplayName) == 0 {
				payload["title"] = fmt.Sprintf("@%s", mUser.Name)
			} else {
				payload["title"] = fmt.Sprintf("@%s", mUser.DisplayName)
			}
			payload["path"] = "/users" + mUser.Name
		}
		body = plain
	} else {
		path, _ := m.repo.GetChannelPath(message.ChannelID)
		payload["title"] = "#" + path
		payload["path"] = "/channels/" + path

		if mUser != nil {
			if len(mUser.DisplayName) == 0 {
				body = fmt.Sprintf("%s: %s", mUser.Name, plain)
			} else {
				body = fmt.Sprintf("%s: %s", mUser.DisplayName, plain)
			}
		} else {
			body = fmt.Sprintf("[ユーザー名が取得できませんでした]: %s", plain)
		}
	}
	if s := utf8string.NewString(body); s.RuneCount() > 100 {
		body = s.Slice(0, 97) + "..."
	}
	payload["body"] = body
	for _, v := range embedded {
		if v.Type == "file" {
			f, _ := m.repo.GetFileMeta(uuid.FromStringOrNil(v.ID))
			if f != nil && f.HasThumbnail {
				payload["image"] = fmt.Sprintf("%s/api/1.0/files/%s/thumbnail", m.origin, v.ID)
				break
			}
		}
	}

	// calculate targets
	targets := map[uuid.UUID]bool{}
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, _ := m.repo.GetUsers()
		for _, v := range users {
			if v.Bot {
				continue
			}
			targets[v.ID] = true
		}

	case !ch.IsPublic: // プライベートチャンネル
		pUsers, _ := m.repo.GetPrivateChannelMemberIDs(message.ChannelID)
		addIDsToSet(targets, pUsers)

	default: // 通常チャンネルメッセージ
		// チャンネル通知ユーザー取得
		users, _ := m.repo.GetSubscribingUserIDs(message.ChannelID)
		addIDsToSet(targets, users)

		// ユーザーグループ・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err == nil {
					addIDsToSet(targets, []uuid.UUID{uid})
				}
			case "group":
				gs, _ := m.repo.GetUserGroupMemberIDs(uuid.FromStringOrNil(v.ID))
				addIDsToSet(targets, gs)
			}
		}

		// ミュート除外
		muted, _ := m.repo.GetMuteUserIDs(ch.ID)
		deleteIDsFromSet(targets, muted)
	}

	// 自分を除外
	delete(targets, message.UserID)

	// send
	for u := range targets {
		devs, _ := m.repo.GetDeviceTokensByUserID(u)
		_ = m.sendToFcm(devs, payload)
	}
}

func (m *FCMManager) sendToFcm(deviceTokens []string, data map[string]string) error {
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
	for _, token := range deviceTokens {
		payload.Token = token
		for i := 0; i < 5; i++ {
			if _, err := m.messaging.Send(context.Background(), payload); err != nil {
				switch {
				case strings.Contains(err.Error(), "registration-token-not-registered"):
					fallthrough
				case strings.Contains(err.Error(), "invalid-argument"):
					if err := m.repo.UnregisterDevice(token); err != nil {
						return err
					}
				case strings.Contains(err.Error(), "internal-error"): // 50x
					if i == 4 {
						return err
					}
					continue // リトライ
				default: // 未知のエラー
					return err
				}
			}
			break
		}
	}
	return nil
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
