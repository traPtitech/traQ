package v3

import (
	"sort"
	"time"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"

	"github.com/gofrs/uuid"
)

type Channel struct {
	ID       uuid.UUID              `json:"id"`
	Name     string                 `json:"name"`
	ParentID optional.Of[uuid.UUID] `json:"parentId"`
	Topic    string                 `json:"topic"`
	Children []uuid.UUID            `json:"children"`
	Archived bool                   `json:"archived"`
	Force    bool                   `json:"force"`
}

func formatChannel(channel *model.Channel, childrenID []uuid.UUID) *Channel {
	return &Channel{
		ID:       channel.ID,
		Name:     channel.Name,
		ParentID: optional.New(channel.ParentID, channel.ParentID != uuid.Nil),
		Topic:    channel.Topic,
		Children: childrenID,
		Archived: channel.IsArchived(),
		Force:    channel.IsForced,
	}
}

type DMChannel struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"userId"`
}

// formatDMChannels ソートされたものを返す
func formatDMChannels(dmcs map[uuid.UUID]uuid.UUID) []*DMChannel {
	res := make([]*DMChannel, 0, len(dmcs))
	for cid, uid := range dmcs {
		res = append(res, &DMChannel{ID: cid, UserID: uid})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ID.String() < res[j].ID.String()
	})
	return res
}

type UserTag struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func formatUserTag(ut model.UserTag) UserTag {
	return UserTag{
		ID:        ut.GetTagID(),
		Tag:       ut.GetTag(),
		IsLocked:  ut.GetIsLocked(),
		CreatedAt: ut.GetCreatedAt(),
		UpdatedAt: ut.GetUpdatedAt(),
	}
}

func formatUserTags(uts []model.UserTag) []UserTag {
	res := make([]UserTag, len(uts))
	for i, ut := range uts {
		res[i] = formatUserTag(ut)
	}
	return res
}

type User struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	IconFileID  uuid.UUID `json:"iconFileId"`
	Bot         bool      `json:"bot"`
	State       int       `json:"state"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// formatUsers ソートされたものを返す
func formatUsers(users []model.UserInfo) []User {
	res := make([]User, len(users))
	for i, user := range users {
		res[i] = User{
			ID:          user.GetID(),
			Name:        user.GetName(),
			DisplayName: user.GetResponseDisplayName(),
			IconFileID:  user.GetIconFileID(),
			Bot:         user.IsBot(),
			State:       user.GetState().Int(),
			UpdatedAt:   user.GetUpdatedAt(),
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ID.String() < res[j].ID.String()
	})
	return res
}

type UserDetail struct {
	ID          uuid.UUID              `json:"id"`
	State       int                    `json:"state"`
	Bot         bool                   `json:"bot"`
	IconFileID  uuid.UUID              `json:"iconFileId"`
	DisplayName string                 `json:"displayName"`
	Name        string                 `json:"name"`
	TwitterID   string                 `json:"twitterId"`
	LastOnline  optional.Of[time.Time] `json:"lastOnline"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Tags        []UserTag              `json:"tags"`
	Groups      []uuid.UUID            `json:"groups"`
	Bio         string                 `json:"bio"`
	HomeChannel optional.Of[uuid.UUID] `json:"homeChannel"`
}

func formatUserDetail(user model.UserInfo, uts []model.UserTag, g []uuid.UUID) *UserDetail {
	return &UserDetail{
		ID:          user.GetID(),
		State:       user.GetState().Int(),
		Bot:         user.IsBot(),
		IconFileID:  user.GetIconFileID(),
		DisplayName: user.GetResponseDisplayName(),
		Name:        user.GetName(),
		TwitterID:   user.GetTwitterID(),
		LastOnline:  user.GetLastOnline(),
		UpdatedAt:   user.GetUpdatedAt(),
		Tags:        formatUserTags(uts),
		Groups:      g,
		Bio:         user.GetBio(),
		HomeChannel: user.GetHomeChannel(),
	}
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

type Bot struct {
	ID              uuid.UUID           `json:"id"`
	BotUserID       uuid.UUID           `json:"botUserId"`
	Description     string              `json:"description"`
	DeveloperID     uuid.UUID           `json:"developerId"`
	SubscribeEvents model.BotEventTypes `json:"subscribeEvents"`
	Mode            model.BotMode       `json:"mode"`
	State           model.BotState      `json:"state"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}

func formatBot(b *model.Bot) *Bot {
	return &Bot{
		ID:              b.ID,
		BotUserID:       b.BotUserID,
		Description:     b.Description,
		SubscribeEvents: b.SubscribeEvents,
		Mode:            b.Mode,
		State:           b.State,
		DeveloperID:     b.CreatorID,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

func formatBots(bs []*model.Bot) []*Bot {
	res := make([]*Bot, len(bs))
	for i, b := range bs {
		res[i] = formatBot(b)
	}
	return res
}

type BotTokens struct {
	VerificationToken string `json:"verificationToken"`
	AccessToken       string `json:"accessToken"`
}

type BotDetail struct {
	ID              uuid.UUID           `json:"id"`
	BotUserID       uuid.UUID           `json:"botUserId"`
	Description     string              `json:"description"`
	DeveloperID     uuid.UUID           `json:"developerId"`
	SubscribeEvents model.BotEventTypes `json:"subscribeEvents"`
	Mode            model.BotMode       `json:"mode"`
	State           model.BotState      `json:"state"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	Tokens          BotTokens           `json:"tokens"`
	Endpoint        string              `json:"endpoint"`
	Privileged      bool                `json:"privileged"`
	Channels        []uuid.UUID         `json:"channels"`
}

func formatBotDetail(b *model.Bot, t *model.OAuth2Token, channels []uuid.UUID) *BotDetail {
	return &BotDetail{
		ID:              b.ID,
		BotUserID:       b.BotUserID,
		Description:     b.Description,
		SubscribeEvents: b.SubscribeEvents,
		Mode:            b.Mode,
		State:           b.State,
		DeveloperID:     b.CreatorID,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
		Tokens: BotTokens{
			VerificationToken: b.VerificationToken,
			AccessToken:       t.AccessToken,
		},
		Endpoint:   b.PostURL,
		Privileged: b.Privileged,
		Channels:   channels,
	}
}

type botEventLogResponse struct {
	RequestID uuid.UUID          `json:"requestId"`
	BotID     uuid.UUID          `json:"botId"`
	Event     model.BotEventType `json:"event"`
	Result    string             `json:"result"`
	Code      int                `json:"code"`
	DateTime  time.Time          `json:"datetime"`
}

func formatBotEventLog(log *model.BotEventLog) *botEventLogResponse {
	return &botEventLogResponse{
		RequestID: log.RequestID,
		BotID:     log.BotID,
		Event:     log.Event,
		Result:    log.Result,
		Code:      log.Code,
		DateTime:  log.DateTime,
	}
}

func formatBotEventLogs(logs []*model.BotEventLog) []*botEventLogResponse {
	res := make([]*botEventLogResponse, len(logs))
	for i, log := range logs {
		res[i] = formatBotEventLog(log)
	}
	return res
}

type Message struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"userId"`
	ChannelID uuid.UUID              `json:"channelId"`
	Content   string                 `json:"content"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Pinned    bool                   `json:"pinned"`
	Stamps    []model.MessageStamp   `json:"stamps"`
	ThreadID  optional.Of[uuid.UUID] `json:"threadId"` // TODO
}

func formatMessage(m *model.Message) *Message {
	return &Message{
		ID:        m.ID,
		UserID:    m.UserID,
		ChannelID: m.ChannelID,
		Content:   m.Text,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Pinned:    m.Pin != nil,
		Stamps:    m.Stamps,
	}
}

type Pin struct {
	UserID   uuid.UUID `json:"userId"`
	PinnedAt time.Time `json:"pinnedAt"`
	Message  *Message  `json:"message"`
}

func formatPin(pin *model.Pin) *Pin {
	res := &Pin{
		UserID:   pin.UserID,
		PinnedAt: pin.CreatedAt,
		Message:  formatMessage(&pin.Message),
	}
	res.Message.Pinned = true
	return res
}

func formatPins(pins []*model.Pin) []*Pin {
	res := make([]*Pin, len(pins))
	for i, p := range pins {
		res[i] = formatPin(p)
	}
	return res
}

type MessagePin struct {
	UserID   uuid.UUID `json:"userId"`
	PinnedAt time.Time `json:"pinnedAt"`
}

func formatMessagePin(pin *model.Pin) *MessagePin {
	return &MessagePin{
		UserID:   pin.UserID,
		PinnedAt: pin.CreatedAt,
	}
}

type MessageClip struct {
	FolderID  uuid.UUID `json:"folderId"`
	ClippedAt time.Time `json:"clippedAt"`
}

func formatMessageClip(cfm *model.ClipFolderMessage) *MessageClip {
	return &MessageClip{
		FolderID:  cfm.FolderID,
		ClippedAt: cfm.CreatedAt,
	}
}

func formatMessageClips(cfms []*model.ClipFolderMessage) []*MessageClip {
	res := make([]*MessageClip, len(cfms))
	for i, cfm := range cfms {
		res[i] = formatMessageClip(cfm)
	}
	return res
}

type UserGroupMember struct {
	ID   uuid.UUID `json:"id"`
	Role string    `json:"role"`
}

// formatUserGroupMembers ソートされたものを返す
func formatUserGroupMembers(members []*model.UserGroupMember) []UserGroupMember {
	arr := make([]UserGroupMember, len(members))
	for i, m := range members {
		arr[i] = UserGroupMember{
			ID:   m.UserID,
			Role: m.Role,
		}
	}

	sort.Slice(arr, func(i, j int) bool { return arr[i].ID.String() < arr[j].ID.String() })
	return arr
}

// formatUserGroupAdmins ソートされたものを返す
func formatUserGroupAdmins(admins []*model.UserGroupAdmin) []uuid.UUID {
	arr := make([]uuid.UUID, len(admins))
	for i, m := range admins {
		arr[i] = m.UserID
	}

	sort.Slice(arr, func(i, j int) bool { return arr[i].String() < arr[j].String() })
	return arr
}

type UserGroup struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Icon        uuid.UUID         `json:"icon"`
	Members     []UserGroupMember `json:"members"`
	Admins      []uuid.UUID       `json:"admins"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func formatUserGroup(g *model.UserGroup) *UserGroup {
	ug := &UserGroup{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Type:        g.Type,
		Icon:        g.Icon,
		Members:     formatUserGroupMembers(g.Members),
		Admins:      formatUserGroupAdmins(g.Admins),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
	return ug
}

// formatUserGroups ソートされたものを返す
func formatUserGroups(gs []*model.UserGroup) []*UserGroup {
	arr := make([]*UserGroup, len(gs))
	for i, g := range gs {
		arr[i] = formatUserGroup(g)
	}

	sort.Slice(arr, func(i, j int) bool { return arr[i].ID.String() < arr[j].ID.String() })
	return arr
}

// FileInfoOldThumbnail deprecated
type FileInfoOldThumbnail struct {
	Mime   string `json:"mime"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type FileInfoThumbnail struct {
	Type   string `json:"type"`
	Mime   string `json:"mime"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type FileInfo struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Mime            string                 `json:"mime"`
	Size            int64                  `json:"size"`
	MD5             string                 `json:"md5"`
	IsAnimatedImage bool                   `json:"isAnimatedImage"`
	CreatedAt       time.Time              `json:"createdAt"`
	Thumbnail       *FileInfoOldThumbnail  `json:"thumbnail"` // deprecated
	ChannelID       optional.Of[uuid.UUID] `json:"channelId"`
	UploaderID      optional.Of[uuid.UUID] `json:"uploaderId"`
	Thumbnails      []FileInfoThumbnail    `json:"thumbnails"`
}

func formatFileInfo(meta model.File) *FileInfo {
	fi := &FileInfo{
		ID:              meta.GetID(),
		Name:            meta.GetFileName(),
		Mime:            meta.GetMIMEType(),
		Size:            meta.GetFileSize(),
		MD5:             meta.GetMD5Hash(),
		IsAnimatedImage: meta.IsAnimatedImage(),
		CreatedAt:       meta.GetCreatedAt(),
		ChannelID:       meta.GetUploadChannelID(),
		UploaderID:      meta.GetCreatorID(),
	}
	if ok, t := meta.GetThumbnail(model.ThumbnailTypeImage); ok {
		fi.Thumbnail = &FileInfoOldThumbnail{
			Mime:   t.Mime,
			Width:  t.Width,
			Height: t.Height,
		}
	}
	ts := meta.GetThumbnails()
	fi.Thumbnails = make([]FileInfoThumbnail, len(ts))
	for i, t := range ts {
		fi.Thumbnails[i] = FileInfoThumbnail{
			Type:   t.Type.String(),
			Mime:   t.Mime,
			Width:  t.Width,
			Height: t.Height,
		}
	}
	return fi
}

func formatFileInfos(metas []model.File) []*FileInfo {
	result := make([]*FileInfo, len(metas))
	for i, meta := range metas {
		result[i] = formatFileInfo(meta)
	}
	return result
}

type OAuth2Client struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	DeveloperID uuid.UUID          `json:"developerId"`
	Scopes      model.AccessScopes `json:"scopes"`
	Confidential bool              `json:"confidential"`
}

func formatOAuth2Client(oc *model.OAuth2Client) *OAuth2Client {
	return &OAuth2Client{
		ID:          oc.ID,
		Name:        oc.Name,
		Description: oc.Description,
		DeveloperID: oc.CreatorID,
		Scopes:      oc.Scopes,
		Confidential: oc.Confidential,
	}
}

func formatOAuth2Clients(ocs []*model.OAuth2Client) []*OAuth2Client {
	arr := make([]*OAuth2Client, len(ocs))
	for i, oc := range ocs {
		arr[i] = formatOAuth2Client(oc)
	}
	return arr
}

type OAuth2ClientDetail struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	DeveloperID uuid.UUID          `json:"developerId"`
	Scopes      model.AccessScopes `json:"scopes"`
	CallbackURL string             `json:"callbackUrl"`
	Secret      string             `json:"secret"`
	Confidential bool               `json:"confidential"`
}

func formatOAuth2ClientDetail(oc *model.OAuth2Client) *OAuth2ClientDetail {
	return &OAuth2ClientDetail{
		ID:          oc.ID,
		Name:        oc.Name,
		Description: oc.Description,
		DeveloperID: oc.CreatorID,
		Scopes:      oc.Scopes,
		CallbackURL: oc.RedirectURI,
		Secret:      oc.Secret,
		Confidential: oc.Confidential,
	}
}

type ClipFolder struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func formatClipFolder(cf *model.ClipFolder) *ClipFolder {
	return &ClipFolder{
		ID:          cf.ID,
		CreatedAt:   cf.CreatedAt,
		Description: cf.Description,
		Name:        cf.Name,
		OwnerID:     cf.OwnerID,
	}
}

// formatClipFolders ソートされたものを返す
func formatClipFolders(cfs []*model.ClipFolder) []*ClipFolder {
	res := make([]*ClipFolder, len(cfs))
	for i, cf := range cfs {
		res[i] = formatClipFolder(cf)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID.String() < res[j].ID.String() })
	return res
}

type ClipFolderMessage struct {
	Message   *Message  `json:"message"`
	ClippedAt time.Time `json:"clippedAt"`
}

func formatClipFolderMessage(cfm *model.ClipFolderMessage) *ClipFolderMessage {
	return &ClipFolderMessage{
		Message:   formatMessage(&cfm.Message),
		ClippedAt: cfm.CreatedAt,
	}
}

func formatClipFolderMessages(cfms []*model.ClipFolderMessage) []*ClipFolderMessage {
	res := make([]*ClipFolderMessage, len(cfms))
	for i, cfm := range cfms {
		res[i] = formatClipFolderMessage(cfm)
	}
	return res
}

type StampPalette struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Stamps      model.UUIDs `json:"stamps"`
	CreatorID   uuid.UUID   `json:"creatorId"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

func formatStampPalette(cf *model.StampPalette) *StampPalette {
	return &StampPalette{
		ID:          cf.ID,
		Name:        cf.Name,
		Description: cf.Description,
		Stamps:      cf.Stamps,
		CreatorID:   cf.CreatorID,
		CreatedAt:   cf.CreatedAt,
		UpdatedAt:   cf.UpdatedAt,
	}
}

// formatStampPalettes ソートされたものを返す
func formatStampPalettes(cfs []*model.StampPalette) []*StampPalette {
	res := make([]*StampPalette, len(cfs))
	for i, cf := range cfs {
		res[i] = formatStampPalette(cf)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID.String() < res[j].ID.String() })
	return res
}
