package router

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"github.com/traPtitech/traQ/utils/validator"
	"golang.org/x/sync/errgroup"
	"image"
	"io"
	"math"
	"mime"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	dmChannelRootUUID  = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))
	pubChannelRootUUID = uuid.Nil
)

type TestRepository struct {
	FS                        storage.FileStorage
	Users                     map[uuid.UUID]model.User
	UsersLock                 sync.RWMutex
	UserGroups                map[uuid.UUID]model.UserGroup
	UserGroupsLock            sync.RWMutex
	UserGroupMembers          map[uuid.UUID]map[uuid.UUID]bool
	UserGroupMembersLock      sync.RWMutex
	Tags                      map[uuid.UUID]model.Tag
	TagsLock                  sync.RWMutex
	UserTags                  map[uuid.UUID]map[uuid.UUID]model.UsersTag
	UserTagsLock              sync.RWMutex
	Channels                  map[uuid.UUID]model.Channel
	ChannelsLock              sync.RWMutex
	ChannelSubscribes         map[uuid.UUID]map[uuid.UUID]bool
	ChannelSubscribesLock     sync.RWMutex
	PrivateChannelMembers     map[uuid.UUID]map[uuid.UUID]bool
	PrivateChannelMembersLock sync.RWMutex
	Messages                  map[uuid.UUID]model.Message
	MessagesLock              sync.RWMutex
	MessageUnreads            map[uuid.UUID]map[uuid.UUID]bool
	MessageUnreadsLock        sync.RWMutex
	MessageReports            []model.MessageReport
	MessageReportsLock        sync.RWMutex
	Pins                      map[uuid.UUID]model.Pin
	PinsLock                  sync.RWMutex
	Stars                     map[uuid.UUID]map[uuid.UUID]bool
	StarsLock                 sync.RWMutex
	Mute                      map[uuid.UUID]map[uuid.UUID]bool
	MuteLock                  sync.RWMutex
	Stamps                    map[uuid.UUID]model.Stamp
	StampsLock                sync.RWMutex
	Files                     map[uuid.UUID]model.File
	FilesLock                 sync.RWMutex
	FilesACL                  map[uuid.UUID]map[uuid.UUID]bool
	FilesACLLock              sync.RWMutex
	Webhooks                  map[uuid.UUID]model.WebhookBot
	WebhooksLock              sync.RWMutex
	OAuth2Clients             map[string]model.OAuth2Client
	OAuth2ClientsLock         sync.RWMutex
	OAuth2Authorizes          map[string]model.OAuth2Authorize
	OAuth2AuthorizesLock      sync.RWMutex
	OAuth2Tokens              map[uuid.UUID]model.OAuth2Token
	OAuth2TokensLock          sync.RWMutex
}

func (repo *TestRepository) GetBotByBotUserID(id uuid.UUID) (*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) UpdateBot(id uuid.UUID, args repository.UpdateBotArgs) error {
	panic("implement me")
}

func (repo *TestRepository) GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error) {
	panic("implement me")
}

func (repo *TestRepository) WriteBotEventLog(log *model.BotEventLog) error {
	panic("implement me")
}

func (repo *TestRepository) GetAllBots() ([]*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) ReissueBotTokens(id uuid.UUID) (*model.Bot, error) {
	panic("implement me")
}

func NewTestRepository() *TestRepository {
	r := &TestRepository{
		FS:                    storage.NewInMemoryFileStorage(),
		Users:                 map[uuid.UUID]model.User{},
		UserGroups:            map[uuid.UUID]model.UserGroup{},
		UserGroupMembers:      map[uuid.UUID]map[uuid.UUID]bool{},
		Tags:                  map[uuid.UUID]model.Tag{},
		UserTags:              map[uuid.UUID]map[uuid.UUID]model.UsersTag{},
		Channels:              map[uuid.UUID]model.Channel{},
		ChannelSubscribes:     map[uuid.UUID]map[uuid.UUID]bool{},
		PrivateChannelMembers: map[uuid.UUID]map[uuid.UUID]bool{},
		Messages:              map[uuid.UUID]model.Message{},
		MessageUnreads:        map[uuid.UUID]map[uuid.UUID]bool{},
		MessageReports:        []model.MessageReport{},
		Pins:                  map[uuid.UUID]model.Pin{},
		Stars:                 map[uuid.UUID]map[uuid.UUID]bool{},
		Mute:                  map[uuid.UUID]map[uuid.UUID]bool{},
		Stamps:                map[uuid.UUID]model.Stamp{},
		Files:                 map[uuid.UUID]model.File{},
		FilesACL:              map[uuid.UUID]map[uuid.UUID]bool{},
		Webhooks:              map[uuid.UUID]model.WebhookBot{},
		OAuth2Clients:         map[string]model.OAuth2Client{},
		OAuth2Authorizes:      map[string]model.OAuth2Authorize{},
		OAuth2Tokens:          map[uuid.UUID]model.OAuth2Token{},
	}
	_, _ = r.CreateUser("traq", "traq", role.Admin)
	return r
}

func (repo *TestRepository) Sync() (bool, error) {
	panic("implement me")
}

func (repo *TestRepository) GetFS() storage.FileStorage {
	return repo.FS
}

func (repo *TestRepository) CreateUser(name, password string, role gorbac.Role) (*model.User, error) {
	repo.UsersLock.Lock()
	defer repo.UsersLock.Unlock()

	for _, v := range repo.Users {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}

	salt := utils.GenerateSalt()
	user := model.User{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		Password:  hex.EncodeToString(utils.HashPassword(password, salt)),
		Salt:      hex.EncodeToString(salt),
		Status:    model.UserAccountStatusActive,
		Bot:       false,
		Role:      role.ID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	iconID, err := repo.GenerateIconFile(user.Name)
	if err != nil {
		return nil, err
	}
	user.Icon = iconID

	repo.Users[user.ID] = user
	return &user, nil
}

func (repo *TestRepository) GetUser(id uuid.UUID) (*model.User, error) {
	repo.UsersLock.RLock()
	u, ok := repo.Users[id]
	repo.UsersLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (repo *TestRepository) GetUserByName(name string) (*model.User, error) {
	repo.UsersLock.RLock()
	defer repo.UsersLock.RUnlock()
	for _, u := range repo.Users {
		u := u
		if u.Name == name {
			return &u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (repo *TestRepository) GetUsers() ([]*model.User, error) {
	repo.UsersLock.RLock()
	result := make([]*model.User, 0, len(repo.Users))
	for _, u := range repo.Users {
		u := u
		result = append(result, &u)
	}
	repo.UsersLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) UserExists(id uuid.UUID) (bool, error) {
	repo.UsersLock.RLock()
	_, ok := repo.Users[id]
	repo.UsersLock.RUnlock()
	return ok, nil
}

func (repo *TestRepository) ChangeUserDisplayName(id uuid.UUID, displayName string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if utf8.RuneCountInString(displayName) > 64 {
		return errors.New("displayName must be <=64 characters")
	}
	repo.UsersLock.Lock()
	u, ok := repo.Users[id]
	if ok {
		u.DisplayName = displayName
		u.UpdatedAt = time.Now()
		repo.Users[id] = u
	}
	repo.UsersLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeUserPassword(id uuid.UUID, password string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	salt := utils.GenerateSalt()
	hashed := utils.HashPassword(password, salt)
	repo.UsersLock.Lock()
	u, ok := repo.Users[id]
	if ok {
		u.Salt = hex.EncodeToString(salt)
		u.Password = hex.EncodeToString(hashed)
		u.UpdatedAt = time.Now()
		repo.Users[id] = u
	}
	repo.UsersLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeUserIcon(id, fileID uuid.UUID) error {
	if id == uuid.Nil || fileID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UsersLock.Lock()
	u, ok := repo.Users[id]
	if ok {
		u.Icon = fileID
		u.UpdatedAt = time.Now()
		repo.Users[id] = u
	}
	repo.UsersLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeUserTwitterID(id uuid.UUID, twitterID string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if err := validator.ValidateVar(twitterID, "twitterid"); err != nil {
		return err
	}
	repo.UsersLock.Lock()
	u, ok := repo.Users[id]
	if ok {
		u.TwitterID = twitterID
		u.UpdatedAt = time.Now()
		repo.Users[id] = u
	}
	repo.UsersLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UsersLock.Lock()
	defer repo.UsersLock.Unlock()
	u, ok := repo.Users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.Status = status
	u.UpdatedAt = time.Now()
	repo.Users[id] = u
	return nil
}

func (repo *TestRepository) UpdateUserLastOnline(id uuid.UUID, t time.Time) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UsersLock.Lock()
	u, ok := repo.Users[id]
	if ok {
		u.LastOnline = &t
		u.UpdatedAt = time.Now()
		repo.Users[id] = u
	}
	repo.UsersLock.Unlock()
	return nil
}

func (repo *TestRepository) IsUserOnline(id uuid.UUID) bool {
	return false
}

func (repo *TestRepository) GetUserLastOnline(id uuid.UUID) (time.Time, error) {
	repo.UsersLock.RLock()
	u, ok := repo.Users[id]
	repo.UsersLock.RUnlock()
	if !ok {
		return time.Time{}, repository.ErrNotFound
	}
	if u.LastOnline != nil {
		return *u.LastOnline, nil
	}
	return time.Time{}, nil
}

func (repo *TestRepository) GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool) {
	panic("implement me")
}

func (repo *TestRepository) UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string) {
	panic("implement me")
}

func (repo *TestRepository) CreateUserGroup(name, description string, adminID uuid.UUID) (*model.UserGroup, error) {
	g := model.UserGroup{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        name,
		Description: description,
		AdminUserID: adminID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}

	repo.UserGroupsLock.Lock()
	defer repo.UserGroupsLock.Unlock()
	for _, v := range repo.UserGroups {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	repo.UserGroups[g.ID] = g
	return &g, nil
}

func (repo *TestRepository) UpdateUserGroup(id uuid.UUID, args repository.UpdateUserGroupNameArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	repo.UserGroupsLock.Lock()
	defer repo.UserGroupsLock.Unlock()
	g, ok := repo.UserGroups[id]
	if !ok {
		return repository.ErrNotFound
	}
	if len(args.Name) > 0 {
		for _, v := range repo.UserGroups {
			if v.Name == args.Name {
				return repository.ErrAlreadyExists
			}
		}
		g.Name = args.Name
	}
	if args.Description.Valid {
		g.Description = args.Description.String
	}
	if args.AdminUserID.Valid {
		g.AdminUserID = args.AdminUserID.UUID
	}
	if err := g.Validate(); err != nil {
		return err
	}
	repo.UserGroups[id] = g
	return nil
}

func (repo *TestRepository) DeleteUserGroup(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UserGroupsLock.Lock()
	defer repo.UserGroupsLock.Unlock()
	repo.UserGroupMembersLock.Lock()
	defer repo.UserGroupMembersLock.Unlock()
	if _, ok := repo.UserGroups[id]; !ok {
		return repository.ErrNotFound
	}
	delete(repo.UserGroups, id)
	delete(repo.UserGroupMembers, id)
	return nil
}

func (repo *TestRepository) GetUserGroup(id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.UserGroupsLock.RLock()
	g, ok := repo.UserGroups[id]
	repo.UserGroupsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &g, nil
}

func (repo *TestRepository) GetUserGroupByName(name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.UserGroupsLock.RLock()
	defer repo.UserGroupsLock.RUnlock()
	for _, v := range repo.UserGroups {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (repo *TestRepository) GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	groups := make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return groups, nil
	}
	repo.UserGroupMembersLock.RLock()
	for gid, users := range repo.UserGroupMembers {
		for uid := range users {
			if uid == userID {
				groups = append(groups, gid)
				break
			}
		}
	}
	repo.UserGroupMembersLock.RUnlock()
	return groups, nil
}

func (repo *TestRepository) GetAllUserGroups() ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	repo.UserGroupsLock.RLock()
	for _, v := range repo.UserGroups {
		v := v
		groups = append(groups, &v)
	}
	repo.UserGroupsLock.RUnlock()
	return groups, nil
}

func (repo *TestRepository) AddUserToGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UserGroupsLock.Lock()
	defer repo.UserGroupsLock.Unlock()
	repo.UserGroupMembersLock.Lock()
	defer repo.UserGroupMembersLock.Unlock()
	g, ok := repo.UserGroups[groupID]
	if !ok {
		return nil
	}
	users, ok := repo.UserGroupMembers[groupID]
	if !ok {
		users = make(map[uuid.UUID]bool)
		repo.UserGroupMembers[groupID] = users
	}
	if !users[userID] {
		users[userID] = true
		g.UpdatedAt = time.Now()
		repo.UserGroups[groupID] = g
	}
	return nil
}

func (repo *TestRepository) RemoveUserFromGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UserGroupsLock.Lock()
	defer repo.UserGroupsLock.Unlock()
	repo.UserGroupMembersLock.Lock()
	defer repo.UserGroupMembersLock.Unlock()
	g, ok := repo.UserGroups[groupID]
	if !ok {
		return nil
	}

	users, ok := repo.UserGroupMembers[groupID]
	if ok && users[userID] {
		delete(users, userID)
		g.UpdatedAt = time.Now()
		repo.UserGroups[groupID] = g
	}
	return nil
}

func (repo *TestRepository) GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	if groupID == uuid.Nil {
		return ids, repository.ErrNotFound
	}
	repo.UserGroupsLock.RLock()
	_, ok := repo.UserGroups[groupID]
	repo.UserGroupsLock.RUnlock()
	if !ok {
		return ids, repository.ErrNotFound
	}
	repo.UserGroupMembersLock.RLock()
	for uid := range repo.UserGroupMembers[groupID] {
		ids = append(ids, uid)
	}
	repo.UserGroupMembersLock.RUnlock()
	return ids, nil
}

func (repo *TestRepository) CreateTag(name string, restricted bool, tagType string) (*model.Tag, error) {
	repo.TagsLock.Lock()
	defer repo.TagsLock.Unlock()
	for _, t := range repo.Tags {
		if t.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	t := model.Tag{
		ID:         uuid.Must(uuid.NewV4()),
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	repo.Tags[t.ID] = t
	return &t, nil
}

func (repo *TestRepository) ChangeTagType(id uuid.UUID, tagType string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.TagsLock.Lock()
	t, ok := repo.Tags[id]
	if ok {
		t.Type = tagType
		t.UpdatedAt = time.Now()
		repo.Tags[id] = t
	}
	repo.TagsLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeTagRestrict(id uuid.UUID, restrict bool) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.TagsLock.Lock()
	t, ok := repo.Tags[id]
	if ok {
		t.Restricted = restrict
		t.UpdatedAt = time.Now()
		repo.Tags[id] = t
	}
	repo.TagsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetAllTags() ([]*model.Tag, error) {
	result := make([]*model.Tag, 0)
	repo.TagsLock.RLock()
	for _, v := range repo.Tags {
		v := v
		result = append(result, &v)
	}
	repo.TagsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetTagByID(id uuid.UUID) (*model.Tag, error) {
	repo.TagsLock.RLock()
	t, ok := repo.Tags[id]
	repo.TagsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &t, nil
}

func (repo *TestRepository) GetTagByName(name string) (*model.Tag, error) {
	repo.TagsLock.RLock()
	defer repo.TagsLock.RUnlock()
	for _, t := range repo.Tags {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (repo *TestRepository) GetOrCreateTagByName(name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.TagsLock.Lock()
	defer repo.TagsLock.Unlock()
	for _, t := range repo.Tags {
		if t.Name == name {
			return &t, nil
		}
	}
	t := model.Tag{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.Tags[t.ID] = t
	return &t, nil
}

func (repo *TestRepository) AddUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	ut := model.UsersTag{
		UserID:    userID,
		TagID:     tagID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.UserTagsLock.Lock()
	tags, ok := repo.UserTags[userID]
	if !ok {
		tags = make(map[uuid.UUID]model.UsersTag)
		repo.UserTags[userID] = tags
	}
	if _, ok := tags[tagID]; ok {
		return repository.ErrAlreadyExists
	}
	tags[tagID] = ut
	repo.UserTagsLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UserTagsLock.Lock()
	defer repo.UserTagsLock.Unlock()
	for id, tag := range repo.UserTags[userID] {
		if id == tagID {
			tag.IsLocked = locked
			tag.UpdatedAt = time.Now()
			repo.UserTags[userID][tagID] = tag
			return nil
		}
	}
	return nil
}

func (repo *TestRepository) DeleteUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.UserTagsLock.Lock()
	tags, ok := repo.UserTags[userID]
	if ok {
		delete(tags, tagID)
	}
	repo.UserTagsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error) {
	repo.UserTagsLock.RLock()
	defer repo.UserTagsLock.RUnlock()
	tags, ok := repo.UserTags[userID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	ut, ok := tags[tagID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	repo.TagsLock.RLock()
	ut.Tag = repo.Tags[ut.TagID]
	repo.TagsLock.RUnlock()
	return &ut, nil
}

func (repo *TestRepository) GetUserTagsByUserID(userID uuid.UUID) ([]*model.UsersTag, error) {
	tags := make([]*model.UsersTag, 0)
	repo.UserTagsLock.RLock()
	for tid, ut := range repo.UserTags[userID] {
		ut := ut
		repo.TagsLock.RLock()
		ut.Tag = repo.Tags[tid]
		repo.TagsLock.RUnlock()
		tags = append(tags, &ut)
	}
	repo.UserTagsLock.RUnlock()
	return tags, nil
}

func (repo *TestRepository) GetUsersByTag(tag string) ([]*model.User, error) {
	users := make([]*model.User, 0)
	repo.TagsLock.RLock()
	tid := uuid.Nil
	for _, t := range repo.Tags {
		if t.Name == tag {
			tid = t.ID
		}
	}
	repo.TagsLock.RUnlock()
	if tid == uuid.Nil {
		return users, nil
	}
	repo.UserTagsLock.RLock()
	for uid, tags := range repo.UserTags {
		if _, ok := tags[tid]; ok {
			repo.UsersLock.RLock()
			u, ok := repo.Users[uid]
			repo.UsersLock.RUnlock()
			if ok {
				users = append(users, &u)
			}
		}
	}
	repo.UserTagsLock.RUnlock()
	return users, nil
}

func (repo *TestRepository) GetUserIDsByTag(tag string) ([]uuid.UUID, error) {
	users := make([]uuid.UUID, 0)
	repo.TagsLock.RLock()
	tid := uuid.Nil
	for _, t := range repo.Tags {
		if t.Name == tag {
			tid = t.ID
		}
	}
	repo.TagsLock.RUnlock()
	if tid == uuid.Nil {
		return users, nil
	}
	repo.UserTagsLock.RLock()
	for uid, tags := range repo.UserTags {
		if _, ok := tags[tid]; ok {
			users = append(users, uid)
		}
	}
	repo.UserTagsLock.RUnlock()
	return users, nil
}

func (repo *TestRepository) GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error) {
	users := make([]uuid.UUID, 0)
	repo.UserTagsLock.RLock()
	for uid, tags := range repo.UserTags {
		if _, ok := tags[tagID]; ok {
			users = append(users, uid)
		}
	}
	repo.UserTagsLock.RUnlock()
	return users, nil
}

func (repo *TestRepository) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, parent); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	switch parent {
	case pubChannelRootUUID: // ルート
		break
	case dmChannelRootUUID: // DMルート
		return nil, repository.ErrForbidden
	default: // ルート以外
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return nil, err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return nil, repository.ErrForbidden
		}

		// 親と公開状況が一致しているか
		if !pCh.IsPublic {
			return nil, repository.ErrForbidden
		}

		// 深さを検証
		for parent, depth := pCh, 2; ; { // 祖先
			if parent.ParentID == uuid.Nil {
				// ルート
				break
			}

			parent, err = repo.GetChannel(parent.ParentID)
			if err != nil {
				if err == repository.ErrNotFound {
					break
				}
				return nil, err
			}
			depth++
			if depth > model.MaxChannelDepth {
				return nil, repository.ErrChannelDepthLimitation
			}
		}
	}

	ch := model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsPublic:  true,
		IsForced:  false,
		IsVisible: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.ChannelsLock.Lock()
	repo.Channels[ch.ID] = ch
	repo.ChannelsLock.Unlock()
	return &ch, nil
}

func (repo *TestRepository) CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error) {
	validMember := make([]uuid.UUID, 0, len(members))
	for _, v := range members {
		ok, err := repo.UserExists(v)
		if err != nil {
			return nil, err
		}
		if ok {
			validMember = append(validMember, v)
		}
	}
	if err := validator.ValidateVar(validMember, "min=1"); err != nil {
		return nil, err
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, uuid.Nil); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	ch := model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsPublic:  false,
		IsForced:  false,
		IsVisible: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.ChannelsLock.Lock()
	repo.Channels[ch.ID] = ch
	for _, v := range validMember {
		_ = repo.AddPrivateChannelMember(ch.ID, v)
	}
	repo.ChannelsLock.Unlock()
	return &ch, nil
}

func (repo *TestRepository) CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error) {
	// ダイレクトメッセージルートの子チャンネルは作れない
	if parentID == dmChannelRootUUID {
		return nil, repository.ErrForbidden
	}

	// 親チャンネル検証
	pCh, err := repo.GetChannel(parentID)
	if err != nil {
		return nil, err
	}

	// ダイレクトメッセージの子チャンネルは作れない
	if pCh.IsDMChannel() {
		return nil, repository.ErrForbidden
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, pCh.ID); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	// 深さを検証
	for parent, depth := pCh, 2; ; { // 祖先
		if parent.ParentID == uuid.Nil {
			// ルート
			break
		}

		parent, err = repo.GetChannel(parent.ParentID)
		if err != nil {
			if err == repository.ErrNotFound {
				break
			}
			return nil, err
		}
		depth++
		if depth > model.MaxChannelDepth {
			return nil, repository.ErrChannelDepthLimitation
		}
	}

	ch := model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		ParentID:  pCh.ID,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsForced:  false,
		IsVisible: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if pCh.IsPublic {
		// 公開チャンネル
		ch.IsPublic = true
		repo.ChannelsLock.Lock()
		repo.Channels[ch.ID] = ch
		repo.ChannelsLock.Unlock()
	} else {
		// 非公開チャンネル
		ch.IsPublic = false

		// 親チャンネルとメンバーは同じ
		ids, err := repo.GetPrivateChannelMemberIDs(pCh.ID)
		if err != nil {
			return nil, err
		}

		repo.ChannelsLock.Lock()
		repo.Channels[ch.ID] = ch
		for _, v := range ids {
			_ = repo.AddPrivateChannelMember(ch.ID, v)
		}
		repo.ChannelsLock.Unlock()
	}
	return &ch, nil
}

func (repo *TestRepository) UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.ChannelsLock.Lock()
	ch, ok := repo.Channels[channelID]
	if ok {
		if visibility != nil {
			ch.IsVisible = *visibility
		}
		if forced != nil {
			ch.IsForced = *forced
		}
		ch.UpdatedAt = time.Now()
		repo.Channels[channelID] = ch
	}
	repo.ChannelsLock.Unlock()
	return nil
}

func (repo *TestRepository) UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.ChannelsLock.Lock()
	ch, ok := repo.Channels[channelID]
	if ok {
		ch.Topic = topic
		ch.UpdatedAt = time.Now()
		repo.Channels[channelID] = ch
	}
	repo.ChannelsLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeChannelName(channelID uuid.UUID, name string) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return err
	}

	// チャンネル取得
	ch, err := repo.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの名前は変更不可能
	if ch.IsDMChannel() {
		return repository.ErrForbidden
	}

	// チャンネル名重複を確認
	if has, err := repo.IsChannelPresent(name, ch.ParentID); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
	}

	// 更新
	repo.ChannelsLock.Lock()
	nch, ok := repo.Channels[channelID]
	if ok {
		nch.Name = name
		nch.UpdatedAt = time.Now()
		repo.Channels[channelID] = nch
	}
	repo.ChannelsLock.Unlock()
	return nil
}

func (repo *TestRepository) ChangeChannelParent(channelID, parent uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル取得
	ch, err := repo.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの親は変更不可能
	if ch.IsDMChannel() {
		return repository.ErrForbidden
	}

	switch parent {
	case uuid.Nil:
		// ルートチャンネル
		break
	case dmChannelRootUUID:
		// DMチャンネルには出来ない
		return repository.ErrForbidden
	default:
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return repository.ErrForbidden
		}

		// 親と公開状況が一致しているか
		if ch.IsPublic != pCh.IsPublic {
			return repository.ErrForbidden
		}

		// 深さを検証
		depth := 1 // ↑で見た親
		for {      // 祖先
			if pCh.ParentID == uuid.Nil {
				// ルート
				break
			}
			if ch.ID == pCh.ID {
				// ループ検出
				return repository.ErrChannelDepthLimitation
			}

			pCh, err = repo.GetChannel(pCh.ParentID)
			if err != nil {
				if err == repository.ErrNotFound {
					break
				}
				return err
			}
			depth++
			if depth >= model.MaxChannelDepth {
				return repository.ErrChannelDepthLimitation
			}
		}
		bottom, err := repo.GetChannelDepth(ch.ID) // 子孫 (自分を含む)
		if err != nil {
			return err
		}
		depth += bottom
		if depth > model.MaxChannelDepth {
			return repository.ErrChannelDepthLimitation
		}
	}

	// チャンネル名検証
	if has, err := repo.IsChannelPresent(ch.Name, parent); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
	}

	// 更新
	repo.ChannelsLock.Lock()
	nch, ok := repo.Channels[channelID]
	if ok {
		nch.ParentID = parent
		nch.UpdatedAt = time.Now()
		repo.Channels[channelID] = nch
	}
	repo.ChannelsLock.Unlock()
	return nil
}

func (repo *TestRepository) DeleteChannel(channelID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	desc, err := repo.GetDescendantChannelIDs(channelID)
	if err != nil {
		return err
	}
	repo.ChannelsLock.Lock()
	for _, id := range append(desc, channelID) {
		delete(repo.Channels, id)
	}
	repo.ChannelsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	repo.ChannelsLock.RLock()
	ch, ok := repo.Channels[channelID]
	repo.ChannelsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &ch, nil
}

func (repo *TestRepository) GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error) {
	repo.MessagesLock.RLock()
	m, ok := repo.Messages[messageID]
	repo.MessagesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	repo.ChannelsLock.RLock()
	ch, ok := repo.Channels[m.ChannelID]
	repo.ChannelsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &ch, nil
}

func (repo *TestRepository) GetChannelsByUserID(userID uuid.UUID) ([]*model.Channel, error) {
	result := make([]*model.Channel, 0)
	repo.ChannelsLock.RLock()
	for _, ch := range repo.Channels {
		ch := ch
		if ch.IsPublic {
			result = append(result, &ch)
		} else if userID != uuid.Nil {
			ok, _ := repo.IsUserPrivateChannelMember(ch.ID, userID)
			if ok {
				result = append(result, &ch)
			}
		}
	}
	repo.ChannelsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	panic("implement me")
}

func (repo *TestRepository) GetAllChannels() ([]*model.Channel, error) {
	repo.ChannelsLock.RLock()
	result := make([]*model.Channel, 0, len(repo.Channels))
	for _, c := range repo.Channels {
		c := c
		result = append(result, &c)
	}
	repo.ChannelsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) IsChannelPresent(name string, parent uuid.UUID) (bool, error) {
	repo.ChannelsLock.RLock()
	defer repo.ChannelsLock.RUnlock()
	for _, ch := range repo.Channels {
		if ch.Name == name && ch.ParentID == parent {
			return true, nil
		}
	}
	return false, nil
}

func (repo *TestRepository) IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	repo.ChannelsLock.RLock()
	ch, ok := repo.Channels[channelID]
	repo.ChannelsLock.RUnlock()
	if !ok {
		return false, nil
	}
	if ch.IsPublic {
		return true, nil
	}
	return repo.IsUserPrivateChannelMember(channelID, userID)
}

func (repo *TestRepository) GetParentChannel(channelID uuid.UUID) (*model.Channel, error) {
	repo.ChannelsLock.RLock()
	defer repo.ChannelsLock.RUnlock()
	ch, ok := repo.Channels[channelID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if ch.ParentID == uuid.Nil {
		return nil, nil
	}
	pCh, ok := repo.Channels[ch.ParentID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &pCh, nil
}

func (repo *TestRepository) GetChildrenChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0)
	repo.ChannelsLock.RLock()
	for cid, ch := range repo.Channels {
		if ch.ParentID == channelID {
			result = append(result, cid)
		}
	}
	repo.ChannelsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetDescendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var descendants []uuid.UUID
	children, err := repo.GetChildrenChannelIDs(channelID)
	if err != nil {
		return nil, err
	}
	descendants = append(descendants, children...)
	for _, v := range children {
		sub, err := repo.GetDescendantChannelIDs(v)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, sub...)
	}
	return descendants, nil
}

func (repo *TestRepository) GetAscendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var ascendants []uuid.UUID
	parent, err := repo.GetParentChannel(channelID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	} else if parent == nil {
		return []uuid.UUID{}, nil
	}
	ascendants = append(ascendants, parent.ID)
	sub, err := repo.GetAscendantChannelIDs(parent.ID)
	if err != nil {
		return nil, err
	}
	ascendants = append(ascendants, sub...)
	return ascendants, nil
}

func (repo *TestRepository) GetChannelPath(id uuid.UUID) (string, error) {
	panic("implement me")
}

func (repo *TestRepository) GetChannelDepth(id uuid.UUID) (int, error) {
	children, err := repo.GetChildrenChannelIDs(id)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, v := range children {
		d, err := repo.GetChannelDepth(v)
		if err != nil {
			return 0, err
		}
		if max < d {
			max = d
		}
	}
	return max + 1, nil
}

func (repo *TestRepository) AddPrivateChannelMember(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.PrivateChannelMembersLock.Lock()
	uids, ok := repo.PrivateChannelMembers[channelID]
	if !ok {
		uids = make(map[uuid.UUID]bool)
	}
	uids[userID] = true
	repo.PrivateChannelMembers[channelID] = uids
	repo.PrivateChannelMembersLock.Unlock()
	return nil
}

func (repo *TestRepository) GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0)
	repo.PrivateChannelMembersLock.RLock()
	for uid := range repo.PrivateChannelMembers[channelID] {
		result = append(result, uid)
	}
	repo.PrivateChannelMembersLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error) {
	repo.PrivateChannelMembersLock.RLock()
	defer repo.PrivateChannelMembersLock.RUnlock()
	uids, ok := repo.PrivateChannelMembers[channelID]
	if !ok {
		return false, nil
	}
	for uid := range uids {
		if userID == uid {
			return true, nil
		}
	}
	return false, nil
}

func (repo *TestRepository) SubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.ChannelSubscribesLock.Lock()
	chMap, ok := repo.ChannelSubscribes[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	repo.ChannelSubscribes[userID] = chMap
	repo.ChannelSubscribesLock.Unlock()
	return nil
}

func (repo *TestRepository) UnsubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.ChannelSubscribesLock.Lock()
	chMap, ok := repo.ChannelSubscribes[userID]
	if ok {
		delete(chMap, channelID)
		repo.ChannelSubscribes[userID] = chMap
	}
	repo.ChannelSubscribesLock.Unlock()
	return nil
}

func (repo *TestRepository) GetSubscribingUserIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	repo.ChannelSubscribesLock.RLock()
	result := make([]uuid.UUID, 0)
	for uid, chMap := range repo.ChannelSubscribes {
		for cid := range chMap {
			if cid == channelID {
				result = append(result, uid)
			}
		}
	}
	repo.ChannelSubscribesLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetSubscribedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	repo.ChannelSubscribesLock.RLock()
	result := make([]uuid.UUID, 0)
	chMap, ok := repo.ChannelSubscribes[userID]
	if ok {
		for id := range chMap {
			result = append(result, id)
		}
	}
	repo.ChannelSubscribesLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	m := &model.Message{
		ID:        uuid.Must(uuid.NewV4()),
		UserID:    userID,
		ChannelID: channelID,
		Text:      text,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	repo.MessagesLock.Lock()
	repo.Messages[m.ID] = *m
	repo.MessagesLock.Unlock()
	return m, nil
}

func (repo *TestRepository) UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	if len(text) == 0 {
		return errors.New("text is empty")
	}

	repo.MessagesLock.Lock()
	defer repo.MessagesLock.Unlock()
	m, ok := repo.Messages[messageID]
	if !ok {
		return repository.ErrNotFound
	}
	m.Text = text
	m.UpdatedAt = time.Now()
	repo.Messages[messageID] = m
	return nil
}

func (repo *TestRepository) DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}

	repo.MessagesLock.Lock()
	defer repo.MessagesLock.Unlock()
	if _, ok := repo.Messages[messageID]; !ok {
		return repository.ErrNotFound
	}
	delete(repo.Messages, messageID)
	return nil
}

func (repo *TestRepository) GetMessageByID(messageID uuid.UUID) (*model.Message, error) {
	repo.MessagesLock.RLock()
	m, ok := repo.Messages[messageID]
	repo.MessagesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &m, nil
}

func (repo *TestRepository) GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) ([]*model.Message, error) {
	tmp := make([]*model.Message, 0)
	repo.MessagesLock.RLock()
	for _, v := range repo.Messages {
		if v.ChannelID == channelID {
			v := v
			tmp = append(tmp, &v)
		}
	}
	repo.MessagesLock.RUnlock()
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].CreatedAt.After(tmp[j].CreatedAt)
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = math.MaxInt32
	}
	result := make([]*model.Message, 0)
	for i := offset; i < len(tmp) && i < offset+limit; i++ {
		result = append(result, tmp[i])
	}
	return result, nil
}

func (repo *TestRepository) GetMessagesByUserID(userID uuid.UUID, limit, offset int) ([]*model.Message, error) {
	tmp := make([]*model.Message, 0)
	repo.MessagesLock.RLock()
	for _, v := range repo.Messages {
		if v.UserID == userID {
			v := v
			tmp = append(tmp, &v)
		}
	}
	repo.MessagesLock.RUnlock()
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].CreatedAt.After(tmp[j].CreatedAt)
	})
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = math.MaxInt32
	}
	result := make([]*model.Message, 0)
	for i := offset; i < len(tmp) && i < offset+limit; i++ {
		result = append(result, tmp[i])
	}
	return result, nil
}

func (repo *TestRepository) GetArchivedMessagesByID(messageID uuid.UUID) ([]*model.ArchivedMessage, error) {
	panic("implement me")
}

func (repo *TestRepository) SetMessageUnread(userID, messageID uuid.UUID) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.MessageUnreadsLock.Lock()
	mMap, ok := repo.MessageUnreads[userID]
	if !ok {
		mMap = make(map[uuid.UUID]bool)
	}
	mMap[messageID] = true
	repo.MessageUnreads[userID] = mMap
	repo.MessageUnreadsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetUnreadMessagesByUserID(userID uuid.UUID) ([]*model.Message, error) {
	result := make([]*model.Message, 0)
	repo.MessageUnreadsLock.RLock()
	repo.MessagesLock.RLock()
	for uid, mMap := range repo.MessageUnreads {
		if uid != userID {
			continue
		}
		for mid := range mMap {
			m, ok := repo.Messages[mid]
			if ok {
				result = append(result, &m)
			}
		}
	}
	repo.MessagesLock.RUnlock()
	repo.MessageUnreadsLock.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		return result[j].CreatedAt.After(result[i].CreatedAt)
	})
	return result, nil
}

func (repo *TestRepository) DeleteUnreadsByMessageID(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.MessageUnreadsLock.Lock()
	for _, mMap := range repo.MessageUnreads {
		var deleted []uuid.UUID
		for mid := range mMap {
			if mid == messageID {
				deleted = append(deleted, mid)
			}
		}
		for _, v := range deleted {
			delete(mMap, v)
		}
	}
	repo.MessageUnreadsLock.Unlock()
	return nil
}

func (repo *TestRepository) DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.MessageUnreadsLock.Lock()
	repo.MessagesLock.RLock()
	for uid, mMap := range repo.MessageUnreads {
		if uid != userID {
			continue
		}
		var deleted []uuid.UUID
		for mid := range mMap {
			m, ok := repo.Messages[mid]
			if ok {
				if m.ChannelID == channelID {
					deleted = append(deleted, mid)
				}
			}
		}
		for _, v := range deleted {
			delete(mMap, v)
		}
	}
	repo.MessagesLock.RUnlock()
	repo.MessageUnreadsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error) {
	panic("implement me")
}

func (repo *TestRepository) CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	if messageID == uuid.Nil || reporterID == uuid.Nil {
		return repository.ErrNilID
	}

	// make report
	report := model.MessageReport{
		ID:        uuid.Must(uuid.NewV4()),
		MessageID: messageID,
		Reporter:  reporterID,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	repo.MessageReportsLock.Lock()
	defer repo.MessageReportsLock.Unlock()
	for _, v := range repo.MessageReports {
		if v.MessageID == messageID && v.Reporter == reporterID {
			return repository.ErrAlreadyExists
		}
	}
	repo.MessageReports = append(repo.MessageReports, report)
	return nil
}

func (repo *TestRepository) GetMessageReports(offset, limit int) ([]*model.MessageReport, error) {
	repo.MessageReportsLock.RLock()
	l := len(repo.MessageReports)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = math.MaxInt32
	}
	result := make([]*model.MessageReport, 0)
	for i := offset; i < l && i < offset+limit; i++ {
		re := repo.MessageReports[i]
		result = append(result, &re)
	}
	repo.MessageReportsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetMessageReportsByMessageID(messageID uuid.UUID) ([]*model.MessageReport, error) {
	repo.MessageReportsLock.RLock()
	result := make([]*model.MessageReport, 0)
	for _, v := range repo.MessageReports {
		if v.MessageID == messageID {
			v := v
			result = append(result, &v)
		}
	}
	repo.MessageReportsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) GetMessageReportsByReporterID(reporterID uuid.UUID) ([]*model.MessageReport, error) {
	repo.MessageReportsLock.RLock()
	result := make([]*model.MessageReport, 0)
	for _, v := range repo.MessageReports {
		if v.Reporter == reporterID {
			v := v
			result = append(result, &v)
		}
	}
	repo.MessageReportsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) AddStampToMessage(messageID, stampID, userID uuid.UUID) (ms *model.MessageStamp, err error) {
	panic("implement me")
}

func (repo *TestRepository) RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error) {
	panic("implement me")
}

func (repo *TestRepository) GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error) {
	return []*model.MessageStamp{}, nil
}

func (repo *TestRepository) GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error) {
	panic("implement me")
}

func (repo *TestRepository) CreateStamp(name string, fileID, userID uuid.UUID) (s *model.Stamp, err error) {
	if fileID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
	}
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	repo.StampsLock.Lock()
	defer repo.StampsLock.Unlock()
	for _, v := range repo.Stamps {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	repo.Stamps[stamp.ID] = *stamp
	return stamp, nil
}

func (repo *TestRepository) UpdateStamp(id uuid.UUID, name string, fileID uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	repo.StampsLock.Lock()
	defer repo.StampsLock.Unlock()
	s, ok := repo.Stamps[id]

	data := map[string]string{}
	if len(name) > 0 {
		if err := validator.ValidateVar(name, "name"); err != nil {
			return err
		}
		s.Name = name
	}
	if fileID != uuid.Nil {
		s.FileID = fileID
	}
	if len(data) == 0 {
		return repository.ErrInvalidArgs
	}

	if !ok {
		return repository.ErrNotFound
	}

	s.UpdatedAt = time.Now()
	repo.Stamps[id] = s
	return nil
}

func (repo *TestRepository) GetStamp(id uuid.UUID) (*model.Stamp, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.StampsLock.RLock()
	s, ok := repo.Stamps[id]
	repo.StampsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &s, nil
}

func (repo *TestRepository) DeleteStamp(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.StampsLock.Lock()
	defer repo.StampsLock.Unlock()
	if _, ok := repo.Stamps[id]; !ok {
		return repository.ErrNotFound
	}
	delete(repo.Stamps, id)
	return nil
}

func (repo *TestRepository) GetAllStamps() (stamps []*model.Stamp, err error) {
	repo.StampsLock.RLock()
	for _, v := range repo.Stamps {
		v := v
		stamps = append(stamps, &v)
	}
	repo.StampsLock.RUnlock()
	return
}

func (repo *TestRepository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	repo.StampsLock.RLock()
	_, ok := repo.Stamps[id]
	repo.StampsLock.RUnlock()
	return ok, nil
}

func (repo *TestRepository) IsStampNameDuplicate(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	repo.StampsLock.RUnlock()
	defer repo.StampsLock.RUnlock()
	for _, v := range repo.Stamps {
		if v.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (repo *TestRepository) GetClipFolder(id uuid.UUID) (*model.ClipFolder, error) {
	panic("implement me")
}

func (repo *TestRepository) GetClipFolders(userID uuid.UUID) ([]*model.ClipFolder, error) {
	panic("implement me")
}

func (repo *TestRepository) CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error) {
	panic("implement me")
}

func (repo *TestRepository) UpdateClipFolderName(id uuid.UUID, name string) error {
	panic("implement me")
}

func (repo *TestRepository) DeleteClipFolder(id uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) GetClipMessage(id uuid.UUID) (*model.Clip, error) {
	panic("implement me")
}

func (repo *TestRepository) GetClipMessages(folderID uuid.UUID) ([]*model.Clip, error) {
	panic("implement me")
}

func (repo *TestRepository) GetClipMessagesByUser(userID uuid.UUID) ([]*model.Clip, error) {
	panic("implement me")
}

func (repo *TestRepository) CreateClip(messageID, folderID, userID uuid.UUID) (*model.Clip, error) {
	panic("implement me")
}

func (repo *TestRepository) ChangeClipFolder(clipID, folderID uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) DeleteClip(id uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) MuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.MuteLock.Lock()
	chMap, ok := repo.Mute[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	repo.Mute[userID] = chMap
	repo.MuteLock.Unlock()
	return nil
}

func (repo *TestRepository) UnmuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.MuteLock.Lock()
	chMap, ok := repo.Mute[userID]
	if ok {
		delete(chMap, channelID)
		repo.Stars[userID] = chMap
	}
	repo.MuteLock.Unlock()
	return nil
}

func (repo *TestRepository) GetMutedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	repo.MuteLock.RLock()
	chMap, ok := repo.Mute[userID]
	if ok {
		for id := range chMap {
			ids = append(ids, id)
		}
	}
	repo.MuteLock.RUnlock()
	return ids, nil
}

func (repo *TestRepository) GetMuteUserIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	repo.MuteLock.RLock()
	for uid, chMap := range repo.Mute {
		if chMap[channelID] {
			ids = append(ids, uid)
		}
	}
	repo.MuteLock.RUnlock()
	return ids, nil
}

func (repo *TestRepository) IsChannelMuted(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	repo.MuteLock.RLock()
	defer repo.MuteLock.RUnlock()
	chMap, ok := repo.Mute[userID]
	if !ok {
		return false, nil
	}
	return chMap[channelID], nil
}

func (repo *TestRepository) AddStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.StarsLock.Lock()
	chMap, ok := repo.Stars[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	repo.Stars[userID] = chMap
	repo.StarsLock.Unlock()
	return nil
}

func (repo *TestRepository) RemoveStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.StarsLock.Lock()
	chMap, ok := repo.Stars[userID]
	if ok {
		delete(chMap, channelID)
		repo.Stars[userID] = chMap
	}
	repo.StarsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetStaredChannels(userID uuid.UUID) ([]uuid.UUID, error) {
	repo.StarsLock.RLock()
	result := make([]uuid.UUID, 0)
	chMap, ok := repo.Stars[userID]
	if ok {
		for id := range chMap {
			result = append(result, id)
		}
	}
	repo.StarsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return uuid.Nil, repository.ErrNilID
	}
	repo.PinsLock.Lock()
	defer repo.PinsLock.Unlock()
	for _, pin := range repo.Pins {
		if pin.MessageID == messageID {
			return pin.ID, nil
		}
	}
	p := model.Pin{
		ID:        uuid.Must(uuid.NewV4()),
		MessageID: messageID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	repo.Pins[p.ID] = p
	return p.ID, nil
}

func (repo *TestRepository) GetPin(id uuid.UUID) (*model.Pin, error) {
	repo.PinsLock.RLock()
	pin, ok := repo.Pins[id]
	repo.PinsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	repo.MessagesLock.RLock()
	pin.Message = repo.Messages[pin.MessageID]
	repo.MessagesLock.RUnlock()
	return &pin, nil
}

func (repo *TestRepository) IsPinned(messageID uuid.UUID) (bool, error) {
	repo.PinsLock.RLock()
	defer repo.PinsLock.RUnlock()
	for _, p := range repo.Pins {
		if p.MessageID == messageID {
			return true, nil
		}
	}
	return false, nil
}

func (repo *TestRepository) DeletePin(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.PinsLock.Lock()
	delete(repo.Pins, id)
	repo.PinsLock.Unlock()
	return nil
}

func (repo *TestRepository) GetPinsByChannelID(channelID uuid.UUID) ([]*model.Pin, error) {
	result := make([]*model.Pin, 0)
	repo.PinsLock.RLock()
	repo.MessagesLock.RLock()
	for _, p := range repo.Pins {
		m, ok := repo.Messages[p.MessageID]
		if ok && m.ChannelID == channelID {
			p := p
			p.Message = m
			result = append(result, &p)
		}
	}
	repo.MessagesLock.RUnlock()
	repo.PinsLock.RUnlock()
	return result, nil
}

func (repo *TestRepository) RegisterDevice(userID uuid.UUID, token string) (*model.Device, error) {
	panic("implement me")
}

func (repo *TestRepository) UnregisterDevice(token string) (err error) {
	panic("implement me")
}

func (repo *TestRepository) GetDevicesByUserID(user uuid.UUID) (result []*model.Device, err error) {
	panic("implement me")
}

func (repo *TestRepository) GetDeviceTokensByUserID(user uuid.UUID) (result []string, err error) {
	panic("implement me")
}

func (repo *TestRepository) GetAllDevices() (result []*model.Device, err error) {
	panic("implement me")
}

func (repo *TestRepository) GetAllDeviceTokens() (result []string, err error) {
	panic("implement me")
}

func (repo *TestRepository) OpenFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := repo.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	rc, err := repo.FS.OpenFileByKey(meta.GetKey())
	return meta, rc, err
}

func (repo *TestRepository) OpenThumbnailFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := repo.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	if meta.HasThumbnail {
		rc, err := repo.FS.OpenFileByKey(meta.GetThumbKey())
		return meta, rc, err
	}
	return meta, nil, repository.ErrNotFound
}

func (repo *TestRepository) GetFileMeta(fileID uuid.UUID) (*model.File, error) {
	if fileID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.FilesLock.RLock()
	meta, ok := repo.Files[fileID]
	repo.FilesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &meta, nil
}

func (repo *TestRepository) DeleteFile(fileID uuid.UUID) error {
	if fileID == uuid.Nil {
		return repository.ErrNilID
	}
	repo.FilesLock.Lock()
	defer repo.FilesLock.Unlock()
	meta, ok := repo.Files[fileID]
	if !ok {
		return repository.ErrNotFound
	}
	delete(repo.Files, fileID)
	return repo.FS.DeleteByKey(meta.GetKey())
}

func (repo *TestRepository) GenerateIconFile(salt string) (uuid.UUID, error) {
	var img bytes.Buffer
	_ = imaging.Encode(&img, utils.GenerateIcon(salt), imaging.PNG)
	file, err := repo.SaveFile(fmt.Sprintf("%s.png", salt), &img, int64(img.Len()), "image/png", model.FileTypeIcon, uuid.Nil)
	return file.ID, err
}

func (repo *TestRepository) SaveFile(name string, src io.Reader, size int64, mimeType string, fType string, creatorID uuid.UUID) (*model.File, error) {
	return repo.SaveFileWithACL(name, src, size, mimeType, fType, creatorID, repository.ACL{uuid.Nil: true})
}

func (repo *TestRepository) SaveFileWithACL(name string, src io.Reader, size int64, mimeType string, fType string, creatorID uuid.UUID, read repository.ACL) (*model.File, error) {
	f := &model.File{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		Size:      size,
		Mime:      mimeType,
		Type:      fType,
		CreatorID: creatorID,
		CreatedAt: time.Now(),
	}
	if len(mimeType) == 0 {
		f.Mime = mime.TypeByExtension(filepath.Ext(name))
		if len(f.Mime) == 0 {
			f.Mime = echo.MIMEOctetStream
		}
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}

	if read != nil {
		read[creatorID] = true
	}

	eg, ctx := errgroup.WithContext(context.Background())

	fileSrc, fileWriter := io.Pipe()
	thumbSrc, thumbWriter := io.Pipe()
	hash := md5.New()

	go func() {
		defer fileWriter.Close()
		defer thumbWriter.Close()
		_, _ = io.Copy(utils.MultiWriter(fileWriter, hash, thumbWriter), src) // 並列化してるけど、pipeじゃなくてbuffer使わないとpipeがブロックしてて意味無い疑惑
	}()

	// fileの保存
	eg.Go(func() error {
		defer fileSrc.Close()
		if err := repo.FS.SaveByKey(fileSrc, f.GetKey(), f.Name, f.Mime, f.Type); err != nil {
			return err
		}
		return nil
	})

	// サムネイルの生成
	eg.Go(func() error {
		// アップロードされたファイルの拡張子が間違えてたり、変なの送ってきた場合
		// サムネイルを生成しないだけで全体のエラーにはしない
		defer thumbSrc.Close()
		size, _ := repo.generateThumbnail(ctx, f, thumbSrc)
		if !size.Empty() {
			f.HasThumbnail = true
			f.ThumbnailWidth = size.Size().X
			f.ThumbnailHeight = size.Size().Y
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	f.Hash = hex.EncodeToString(hash.Sum(nil))
	repo.FilesLock.Lock()
	repo.FilesACLLock.Lock()
	repo.Files[f.ID] = *f
	repo.FilesACL[f.ID] = read
	repo.FilesACLLock.Unlock()
	repo.FilesLock.Unlock()
	return f, nil
}

func (repo *TestRepository) RegenerateThumbnail(fileID uuid.UUID) (bool, error) {
	return false, nil
}

func (repo *TestRepository) IsFileAccessible(fileID, userID uuid.UUID) (bool, error) {
	if fileID == uuid.Nil {
		return false, repository.ErrNilID
	}
	repo.FilesLock.RLock()
	_, ok := repo.Files[fileID]
	repo.FilesLock.RUnlock()
	if !ok {
		return false, repository.ErrNotFound
	}

	var allow bool
	repo.FilesACLLock.RLock()
	defer repo.FilesACLLock.RUnlock()
	for uid, a := range repo.FilesACL[fileID] {
		if uid == uuid.Nil || uid == userID {
			if a {
				allow = true
			} else {
				return false, nil
			}
		}
	}
	return allow, nil
}

func (repo *TestRepository) generateThumbnail(ctx context.Context, f *model.File, src io.Reader) (image.Rectangle, error) {
	orig, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return image.ZR, err
	}

	img := imaging.Fit(orig, 360, 480, imaging.Linear)

	r, w := io.Pipe()
	go func() {
		_ = imaging.Encode(w, img, imaging.PNG)
		_ = w.Close()
	}()

	if err := repo.FS.SaveByKey(r, f.GetThumbKey(), f.GetThumbKey()+".png", "image/png", model.FileTypeThumbnail); err != nil {
		return image.ZR, err
	}
	return img.Bounds(), nil
}

func (repo *TestRepository) CreateWebhook(name, description string, channelID, creatorID uuid.UUID, secret string) (model.Webhook, error) {
	if len(name) == 0 || utf8.RuneCountInString(name) > 32 {
		return nil, errors.New("invalid name")
	}
	uid := uuid.Must(uuid.NewV4())
	bid := uuid.Must(uuid.NewV4())
	iconID, err := repo.GenerateIconFile(name)
	if err != nil {
		return nil, err
	}

	u := model.User{
		ID:          uid,
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(uid.Bytes()),
		DisplayName: name,
		Icon:        iconID,
		Bot:         true,
		Status:      model.UserAccountStatusActive,
		Role:        role.Bot.ID(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	wb := model.WebhookBot{
		ID:          bid,
		BotUserID:   uid,
		Description: description,
		Secret:      secret,
		ChannelID:   channelID,
		CreatorID:   creatorID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	repo.WebhooksLock.Lock()
	repo.UsersLock.Lock()
	repo.Users[uid] = u
	repo.Webhooks[bid] = wb
	repo.UsersLock.Unlock()
	repo.WebhooksLock.Unlock()

	wb.BotUser = u
	return &wb, nil
}

func (repo *TestRepository) UpdateWebhook(id uuid.UUID, args repository.UpdateWebhookArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	repo.WebhooksLock.Lock()
	repo.UsersLock.Lock()
	defer repo.WebhooksLock.Unlock()
	defer repo.UsersLock.Unlock()
	wb, ok := repo.Webhooks[id]
	if !ok {
		return repository.ErrNotFound
	}
	u := repo.Users[wb.GetBotUserID()]

	if args.Description.Valid {
		wb.Description = args.Description.String
		wb.UpdatedAt = time.Now()
	}
	if args.ChannelID.Valid {
		wb.ChannelID = args.ChannelID.UUID
		wb.UpdatedAt = time.Now()
	}
	if args.Secret.Valid {
		wb.Secret = args.Secret.String
		wb.UpdatedAt = time.Now()
	}
	if args.Name.Valid {
		if len(args.Name.String) == 0 || utf8.RuneCountInString(args.Name.String) > 32 {
			return errors.New("invalid name")
		}
		u.DisplayName = args.Name.String
		u.UpdatedAt = time.Now()
	}

	repo.Webhooks[id] = wb
	repo.Users[u.ID] = u
	return nil
}

func (repo *TestRepository) DeleteWebhook(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	repo.WebhooksLock.Lock()
	repo.UsersLock.Lock()
	defer repo.WebhooksLock.Unlock()
	defer repo.UsersLock.Unlock()
	wb, ok := repo.Webhooks[id]
	if !ok {
		return repository.ErrNotFound
	}
	delete(repo.Webhooks, id)
	u := repo.Users[wb.BotUserID]
	u.Status = model.UserAccountStatusDeactivated
	u.UpdatedAt = time.Now()
	repo.Users[wb.BotUserID] = u
	return nil
}

func (repo *TestRepository) GetWebhook(id uuid.UUID) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.WebhooksLock.RLock()
	repo.UsersLock.RLock()
	defer repo.WebhooksLock.RUnlock()
	defer repo.UsersLock.RUnlock()
	w, ok := repo.Webhooks[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	w.BotUser = repo.Users[w.BotUserID]
	return &w, nil
}

func (repo *TestRepository) GetWebhookByBotUserID(id uuid.UUID) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.WebhooksLock.RLock()
	repo.UsersLock.RLock()
	defer repo.WebhooksLock.RUnlock()
	defer repo.UsersLock.RUnlock()
	var (
		w  model.WebhookBot
		ok bool
	)
	for _, v := range repo.Webhooks {
		if v.BotUserID == id {
			w = v
			ok = true
			break
		}
	}
	if !ok {
		return nil, repository.ErrNotFound
	}
	w.BotUser = repo.Users[w.BotUserID]
	return &w, nil
}

func (repo *TestRepository) GetAllWebhooks() ([]model.Webhook, error) {
	arr := make([]model.Webhook, 0)
	repo.WebhooksLock.RLock()
	repo.UsersLock.RLock()
	for _, v := range repo.Webhooks {
		v := v
		v.BotUser = repo.Users[v.BotUserID]
		arr = append(arr, &v)
	}
	repo.UsersLock.RUnlock()
	repo.WebhooksLock.RUnlock()
	return arr, nil
}

func (repo *TestRepository) GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error) {
	arr := make([]model.Webhook, 0)
	if creatorID == uuid.Nil {
		return arr, nil
	}
	repo.WebhooksLock.RLock()
	repo.UsersLock.RLock()
	for _, v := range repo.Webhooks {
		if v.CreatorID == creatorID {
			v := v
			v.BotUser = repo.Users[v.BotUserID]
			arr = append(arr, &v)
		}
	}
	repo.UsersLock.RUnlock()
	repo.WebhooksLock.RUnlock()
	return arr, nil
}

func (repo *TestRepository) GetClient(id string) (*model.OAuth2Client, error) {
	if len(id) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.OAuth2ClientsLock.RLock()
	c, ok := repo.OAuth2Clients[id]
	repo.OAuth2ClientsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &c, nil
}

func (repo *TestRepository) GetClientsByUser(userID uuid.UUID) ([]*model.OAuth2Client, error) {
	cs := make([]*model.OAuth2Client, 0)
	if userID == uuid.Nil {
		return cs, nil
	}
	repo.OAuth2ClientsLock.RLock()
	for _, v := range repo.OAuth2Clients {
		v := v
		if v.CreatorID == userID {
			cs = append(cs, &v)
		}
	}
	repo.OAuth2ClientsLock.RUnlock()
	return cs, nil
}

func (repo *TestRepository) SaveClient(client *model.OAuth2Client) error {
	repo.OAuth2ClientsLock.Lock()
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()
	repo.OAuth2Clients[client.ID] = *client
	repo.OAuth2ClientsLock.Unlock()
	return nil
}

func (repo *TestRepository) UpdateClient(client *model.OAuth2Client) error {
	if len(client.ID) == 0 {
		return repository.ErrNilID
	}
	repo.OAuth2ClientsLock.Lock()
	defer repo.OAuth2ClientsLock.Unlock()
	c, ok := repo.OAuth2Clients[client.ID]
	if ok {
		c.UpdatedAt = time.Now()
		c.Name = client.Name
		c.Description = client.Description
		c.Confidential = client.Confidential
		c.CreatorID = client.CreatorID
		c.Secret = client.Secret
		c.RedirectURI = client.RedirectURI
		c.Scopes = client.Scopes
		repo.OAuth2Clients[client.ID] = c
	}
	return nil
}

func (repo *TestRepository) DeleteClient(id string) error {
	if len(id) == 0 {
		return nil
	}
	repo.OAuth2ClientsLock.Lock()
	repo.OAuth2AuthorizesLock.Lock()
	repo.OAuth2TokensLock.Lock()
	targetT := make([]uuid.UUID, 0)
	for k, v := range repo.OAuth2Tokens {
		if v.ClientID == id {
			targetT = append(targetT, k)
		}
	}
	for _, v := range targetT {
		delete(repo.OAuth2Tokens, v)
	}
	targetA := make([]string, 0)
	for k, v := range repo.OAuth2Authorizes {
		if v.ClientID == id {
			targetA = append(targetA, k)
		}
	}
	for _, v := range targetA {
		delete(repo.OAuth2Authorizes, v)
	}
	delete(repo.OAuth2Clients, id)
	repo.OAuth2TokensLock.Unlock()
	repo.OAuth2AuthorizesLock.Unlock()
	repo.OAuth2ClientsLock.Unlock()
	return nil
}

func (repo *TestRepository) SaveAuthorize(data *model.OAuth2Authorize) error {
	repo.OAuth2AuthorizesLock.Lock()
	data.CreatedAt = time.Now()
	repo.OAuth2Authorizes[data.Code] = *data
	repo.OAuth2AuthorizesLock.Unlock()
	return nil
}

func (repo *TestRepository) GetAuthorize(code string) (*model.OAuth2Authorize, error) {
	if len(code) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.OAuth2AuthorizesLock.RLock()
	a, ok := repo.OAuth2Authorizes[code]
	repo.OAuth2AuthorizesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &a, nil
}

func (repo *TestRepository) DeleteAuthorize(code string) error {
	if len(code) == 0 {
		return nil
	}
	repo.OAuth2AuthorizesLock.Lock()
	delete(repo.OAuth2Authorizes, code)
	repo.OAuth2AuthorizesLock.Unlock()
	return nil
}

func (repo *TestRepository) IssueToken(client *model.OAuth2Client, userID uuid.UUID, redirectURI string, scope model.AccessScopes, expire int, refresh bool) (*model.OAuth2Token, error) {
	newToken := &model.OAuth2Token{
		ID:             uuid.Must(uuid.NewV4()),
		UserID:         userID,
		RedirectURI:    redirectURI,
		AccessToken:    utils.RandAlphabetAndNumberString(36),
		RefreshToken:   utils.RandAlphabetAndNumberString(36),
		RefreshEnabled: refresh,
		CreatedAt:      time.Now(),
		ExpiresIn:      expire,
		Scopes:         scope,
	}

	if client != nil {
		newToken.ClientID = client.ID
	}

	repo.OAuth2TokensLock.Lock()
	repo.OAuth2Tokens[newToken.ID] = *newToken
	repo.OAuth2TokensLock.Unlock()
	return newToken, nil
}

func (repo *TestRepository) GetTokenByID(id uuid.UUID) (*model.OAuth2Token, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	repo.OAuth2TokensLock.RLock()
	t, ok := repo.OAuth2Tokens[id]
	repo.OAuth2TokensLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &t, nil
}

func (repo *TestRepository) DeleteTokenByID(id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	repo.OAuth2TokensLock.Lock()
	delete(repo.OAuth2Tokens, id)
	repo.OAuth2TokensLock.Unlock()
	return nil
}

func (repo *TestRepository) GetTokenByAccess(access string) (*model.OAuth2Token, error) {
	if len(access) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.OAuth2TokensLock.RLock()
	defer repo.OAuth2TokensLock.RUnlock()
	for _, v := range repo.OAuth2Tokens {
		if v.AccessToken == access {
			return &v, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (repo *TestRepository) DeleteTokenByAccess(access string) error {
	if len(access) == 0 {
		return nil
	}
	repo.OAuth2TokensLock.Lock()
	defer repo.OAuth2TokensLock.Unlock()
	for k, v := range repo.OAuth2Tokens {
		if v.AccessToken == access {
			delete(repo.OAuth2Tokens, k)
			return nil
		}
	}
	return nil
}

func (repo *TestRepository) GetTokenByRefresh(refresh string) (*model.OAuth2Token, error) {
	if len(refresh) == 0 {
		return nil, repository.ErrNotFound
	}
	repo.OAuth2TokensLock.RLock()
	defer repo.OAuth2TokensLock.RUnlock()
	for _, v := range repo.OAuth2Tokens {
		if v.RefreshEnabled && v.RefreshToken == refresh {
			return &v, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (repo *TestRepository) DeleteTokenByRefresh(refresh string) error {
	if len(refresh) == 0 {
		return nil
	}
	repo.OAuth2TokensLock.Lock()
	defer repo.OAuth2TokensLock.Unlock()
	for k, v := range repo.OAuth2Tokens {
		if v.RefreshEnabled && v.RefreshToken == refresh {
			delete(repo.OAuth2Tokens, k)
			return nil
		}
	}
	return nil
}

func (repo *TestRepository) GetTokensByUser(userID uuid.UUID) ([]*model.OAuth2Token, error) {
	ts := make([]*model.OAuth2Token, 0)
	if userID == uuid.Nil {
		return ts, nil
	}
	repo.OAuth2TokensLock.RLock()
	for _, v := range repo.OAuth2Tokens {
		v := v
		if v.UserID == userID {
			ts = append(ts, &v)
		}
	}
	repo.OAuth2TokensLock.RUnlock()
	return ts, nil
}

func (repo *TestRepository) DeleteTokenByUser(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return nil
	}
	repo.OAuth2TokensLock.Lock()
	target := make([]uuid.UUID, 0)
	for k, v := range repo.OAuth2Tokens {
		if v.UserID == userID {
			target = append(target, k)
		}
	}
	for _, v := range target {
		delete(repo.OAuth2Tokens, v)
	}
	repo.OAuth2TokensLock.Unlock()
	return nil
}

func (repo *TestRepository) DeleteTokenByClient(clientID string) error {
	if len(clientID) == 0 {
		return nil
	}
	repo.OAuth2TokensLock.Lock()
	target := make([]uuid.UUID, 0)
	for k, v := range repo.OAuth2Tokens {
		if v.ClientID == clientID {
			target = append(target, k)
		}
	}
	for _, v := range target {
		delete(repo.OAuth2Tokens, v)
	}
	repo.OAuth2TokensLock.Unlock()
	return nil
}

func (repo *TestRepository) CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error {
	panic("implement me")
}

func (repo *TestRepository) GetBotByID(id uuid.UUID) (*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) GetBotByCode(code string) (*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error) {
	panic("implement me")
}

func (repo *TestRepository) ChangeBotState(id uuid.UUID, state model.BotState) error {
	panic("implement me")
}

func (repo *TestRepository) DeleteBot(id uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) AddBotToChannel(botID, channelID uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) RemoveBotFromChannel(botID, channelID uuid.UUID) error {
	panic("implement me")
}

func (repo *TestRepository) GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error) {
	panic("implement me")
}
