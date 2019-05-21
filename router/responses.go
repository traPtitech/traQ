package router

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

type userResponse struct {
	UserID      uuid.UUID  `json:"userId"`
	Name        string     `json:"name"`
	DisplayName string     `json:"displayName"`
	IconID      uuid.UUID  `json:"iconFileId"`
	Bot         bool       `json:"bot"`
	TwitterID   string     `json:"twitterId"`
	LastOnline  *time.Time `json:"lastOnline"`
	IsOnline    bool       `json:"isOnline"`
	Suspended   bool       `json:"suspended"`
	Status      int        `json:"accountStatus"`
}

func (h *Handlers) formatUser(user *model.User) *userResponse {
	res := &userResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    h.Repo.IsUserOnline(user.ID),
		Suspended:   user.Status != model.UserAccountStatusActive,
		Status:      int(user.Status),
	}
	if t, err := h.Repo.GetUserLastOnline(user.ID); err == nil && !t.IsZero() {
		res.LastOnline = &t
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res
}

func (h *Handlers) formatUsers(users []*model.User) []*userResponse {
	res := make([]*userResponse, len(users))
	for i, user := range users {
		res[i] = h.formatUser(user)
	}
	return res
}

type userDetailResponse struct {
	UserID      uuid.UUID      `json:"userId"`
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	IconID      uuid.UUID      `json:"iconFileId"`
	Bot         bool           `json:"bot"`
	TwitterID   string         `json:"twitterId"`
	LastOnline  *time.Time     `json:"lastOnline"`
	IsOnline    bool           `json:"isOnline"`
	Suspended   bool           `json:"suspended"`
	Status      int            `json:"accountStatus"`
	TagList     []*tagResponse `json:"tagList"`
}

func (h *Handlers) formatUserDetail(user *model.User, tagList []*model.UsersTag) (*userDetailResponse, error) {
	res := &userDetailResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    h.Repo.IsUserOnline(user.ID),
		Suspended:   user.Status != model.UserAccountStatusActive,
		Status:      int(user.Status),
		TagList:     formatTags(tagList),
	}
	if t, err := h.Repo.GetUserLastOnline(user.ID); err == nil && !t.IsZero() {
		res.LastOnline = &t
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res, nil
}

type messageResponse struct {
	MessageID       uuid.UUID            `json:"messageId"`
	UserID          uuid.UUID            `json:"userId"`
	ParentChannelID uuid.UUID            `json:"parentChannelId"`
	Content         string               `json:"content"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
	Pin             bool                 `json:"pin"`
	Reported        bool                 `json:"reported"`
	StampList       []model.MessageStamp `json:"stampList"`
}

func formatMessage(m *model.Message) *messageResponse {
	return &messageResponse{
		MessageID:       m.ID,
		UserID:          m.UserID,
		ParentChannelID: m.ChannelID,
		Pin:             m.Pin != nil,
		Content:         m.Text,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		StampList:       m.Stamps,
	}
}

func formatMessages(ms []*model.Message) []*messageResponse {
	res := make([]*messageResponse, len(ms))
	for i, m := range ms {
		res[i] = formatMessage(m)
	}
	return res
}

type pinResponse struct {
	PinID     uuid.UUID        `json:"pinId"`
	ChannelID uuid.UUID        `json:"channelId"`
	UserID    uuid.UUID        `json:"userId"`
	DateTime  time.Time        `json:"dateTime"`
	Message   *messageResponse `json:"message"`
}

func formatPin(pin *model.Pin) *pinResponse {
	res := &pinResponse{
		PinID:     pin.ID,
		ChannelID: pin.Message.ChannelID,
		UserID:    pin.UserID,
		DateTime:  pin.CreatedAt,
		Message:   formatMessage(&pin.Message),
	}
	res.Message.Pin = true
	return res
}

func formatPins(pins []*model.Pin) []*pinResponse {
	res := make([]*pinResponse, len(pins))
	for i, p := range pins {
		res[i] = formatPin(p)
	}
	return res
}

type webhookResponse struct {
	WebhookID   string    `json:"webhookId"`
	BotUserID   string    `json:"botUserId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Secure      bool      `json:"secure"`
	ChannelID   string    `json:"channelId"`
	CreatorID   string    `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func formatWebhook(w model.Webhook) *webhookResponse {
	return &webhookResponse{
		WebhookID:   w.GetID().String(),
		BotUserID:   w.GetBotUserID().String(),
		DisplayName: w.GetName(),
		Description: w.GetDescription(),
		Secure:      len(w.GetSecret()) > 0,
		ChannelID:   w.GetChannelID().String(),
		CreatorID:   w.GetCreatorID().String(),
		CreatedAt:   w.GetCreatedAt(),
		UpdatedAt:   w.GetUpdatedAt(),
	}
}

func formatWebhooks(ws []model.Webhook) []*webhookResponse {
	res := make([]*webhookResponse, len(ws))
	for i, w := range ws {
		res[i] = formatWebhook(w)
	}
	return res
}

type channelResponse struct {
	ChannelID  string      `json:"channelId"`
	Name       string      `json:"name"`
	Parent     string      `json:"parent"`
	Topic      string      `json:"topic"`
	Children   []uuid.UUID `json:"children"`
	Member     []uuid.UUID `json:"member"`
	Visibility bool        `json:"visibility"`
	Force      bool        `json:"force"`
	Private    bool        `json:"private"`
	DM         bool        `json:"dm"`
}

func (h *Handlers) formatChannel(channel *model.Channel) (response *channelResponse, err error) {
	response = &channelResponse{
		ChannelID:  channel.ID.String(),
		Name:       channel.Name,
		Topic:      channel.Topic,
		Visibility: channel.IsVisible,
		Force:      channel.IsForced,
		Private:    !channel.IsPublic,
		DM:         channel.IsDMChannel(),
		Member:     make([]uuid.UUID, 0),
	}
	if channel.ParentID != uuid.Nil {
		response.Parent = channel.ParentID.String()
	}
	response.Children, err = h.Repo.GetChildrenChannelIDs(channel.ID)
	if err != nil {
		return nil, err
	}

	if response.Private {
		response.Member, err = h.Repo.GetPrivateChannelMemberIDs(channel.ID)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

type botResponse struct {
	BotID           uuid.UUID       `json:"botId"`
	BotUserID       uuid.UUID       `json:"botUserId"`
	Description     string          `json:"description"`
	SubscribeEvents model.BotEvents `json:"subscribeEvents"`
	State           model.BotState  `json:"state"`
	CreatorID       uuid.UUID       `json:"creatorId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func formatBot(b *model.Bot) *botResponse {
	return &botResponse{
		BotID:           b.ID,
		BotUserID:       b.BotUserID,
		Description:     b.Description,
		SubscribeEvents: b.SubscribeEvents,
		State:           b.State,
		CreatorID:       b.CreatorID,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

func formatBots(bs []*model.Bot) []*botResponse {
	res := make([]*botResponse, len(bs))
	for i, b := range bs {
		res[i] = formatBot(b)
	}
	return res
}

type botDetailResponse struct {
	BotID            uuid.UUID       `json:"botId"`
	BotUserID        uuid.UUID       `json:"botUserId"`
	Description      string          `json:"description"`
	SubscribeEvents  model.BotEvents `json:"subscribeEvents"`
	State            model.BotState  `json:"state"`
	CreatorID        uuid.UUID       `json:"creatorId"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	VerificationCode string          `json:"verificationCode"`
	AccessToken      string          `json:"accessToken"`
	PostURL          string          `json:"postUrl"`
	Privileged       bool            `json:"privileged"`
	BotCode          string          `json:"botCode"`
}

func formatBotDetail(b *model.Bot, t *model.OAuth2Token) *botDetailResponse {
	return &botDetailResponse{
		BotID:            b.ID,
		BotUserID:        b.BotUserID,
		Description:      b.Description,
		SubscribeEvents:  b.SubscribeEvents,
		State:            b.State,
		CreatorID:        b.CreatorID,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		VerificationCode: b.VerificationToken,
		AccessToken:      t.AccessToken,
		PostURL:          b.PostURL,
		Privileged:       b.Privileged,
		BotCode:          b.BotCode,
	}
}

type tagResponse struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func formatTag(ut *model.UsersTag) *tagResponse {
	return &tagResponse{
		ID:        ut.Tag.ID,
		Tag:       ut.Tag.Name,
		IsLocked:  ut.IsLocked,
		CreatedAt: ut.CreatedAt,
		UpdatedAt: ut.UpdatedAt,
	}
}

func formatTags(uts []*model.UsersTag) []*tagResponse {
	res := make([]*tagResponse, len(uts))
	for i, ut := range uts {
		res[i] = formatTag(ut)
	}
	return res
}

type userGroupResponse struct {
	GroupID     uuid.UUID   `json:"groupId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	AdminUserID uuid.UUID   `json:"adminUserId"`
	Members     []uuid.UUID `json:"members"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

func (h *Handlers) formatUserGroup(g *model.UserGroup) (r *userGroupResponse, err error) {
	r = &userGroupResponse{
		GroupID:     g.ID,
		Name:        g.Name,
		Description: g.Description,
		Type:        g.Type,
		AdminUserID: g.AdminUserID,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
	r.Members, err = h.Repo.GetUserGroupMemberIDs(g.ID)
	return
}

func (h *Handlers) formatUserGroups(gs []*model.UserGroup) ([]*userGroupResponse, error) {
	arr := make([]*userGroupResponse, len(gs))
	for i, g := range gs {
		r, err := h.formatUserGroup(g)
		if err != nil {
			return nil, err
		}
		arr[i] = r
	}
	return arr, nil
}
