package v3

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

type Channel struct {
	ID         uuid.UUID     `json:"id"`
	Name       string        `json:"name"`
	ParentID   uuid.NullUUID `json:"parentId"`
	Topic      string        `json:"topic"`
	Children   []uuid.UUID   `json:"children"`
	Visibility bool          `json:"visibility"`
	Force      bool          `json:"force"`
}

func formatChannel(channel *model.Channel, childrenID []uuid.UUID) *Channel {
	return &Channel{
		ID:         channel.ID,
		Name:       channel.Name,
		ParentID:   uuid.NullUUID{UUID: channel.ParentID, Valid: channel.ParentID != uuid.Nil},
		Topic:      channel.Topic,
		Children:   childrenID,
		Visibility: channel.IsVisible,
		Force:      channel.IsForced,
	}
}

type UserTag struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func formatUserTags(uts []*model.UsersTag) []UserTag {
	res := make([]UserTag, len(uts))
	for i, ut := range uts {
		res[i] = UserTag{
			ID:        ut.Tag.ID,
			Tag:       ut.Tag.Name,
			IsLocked:  ut.IsLocked,
			CreatedAt: ut.CreatedAt,
			UpdatedAt: ut.UpdatedAt,
		}
	}
	return res
}

type UserDetail struct {
	ID          uuid.UUID   `json:"id"`
	State       int         `json:"state"`
	Bot         bool        `json:"bot"`
	IconFileID  uuid.UUID   `json:"iconFileId"`
	DisplayName string      `json:"displayName"`
	Name        string      `json:"name"`
	TwitterID   string      `json:"twitterId"`
	LastOnline  *time.Time  `json:"lastOnline"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Tags        []UserTag   `json:"tags"`
	Groups      []uuid.UUID `json:"groups"`
	Bio         string      `json:"bio"`
}

func formatUserDetail(user *model.User, uts []*model.UsersTag, g []uuid.UUID) *UserDetail {
	u := &UserDetail{
		ID:          user.ID,
		State:       user.Status.Int(),
		Bot:         user.Bot,
		IconFileID:  user.Icon,
		DisplayName: user.DisplayName,
		Name:        user.Name,
		TwitterID:   user.TwitterID,
		LastOnline:  user.LastOnline.Ptr(),
		UpdatedAt:   user.UpdatedAt,
		Tags:        formatUserTags(uts),
		Groups:      g,
		Bio:         "", // TODO
	}

	if len(u.DisplayName) == 0 {
		u.DisplayName = u.Name
	}
	return u
}

type Webhook struct {
	WebhookID   string    `json:"id"`
	BotUserID   string    `json:"botUserId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Secure      bool      `json:"secure"`
	ChannelID   string    `json:"channelId"`
	OwnerID     string    `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func formatWebhook(w model.Webhook) *Webhook {
	return &Webhook{
		WebhookID:   w.GetID().String(),
		BotUserID:   w.GetBotUserID().String(),
		DisplayName: w.GetName(),
		Description: w.GetDescription(),
		Secure:      len(w.GetSecret()) > 0,
		ChannelID:   w.GetChannelID().String(),
		OwnerID:     w.GetCreatorID().String(),
		CreatedAt:   w.GetCreatedAt(),
		UpdatedAt:   w.GetUpdatedAt(),
	}
}

func formatWebhooks(ws []model.Webhook) []*Webhook {
	res := make([]*Webhook, len(ws))
	for i, w := range ws {
		res[i] = formatWebhook(w)
	}
	return res
}
