package router

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/labstack/echo"
	"github.com/mikespook/gorbac"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"github.com/traPtitech/traQ/utils/thumb"
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
}

func NewTestRepository() *TestRepository {
	r := &TestRepository{
		FS:                    storage.NewInMemoryFileStorage(),
		Users:                 make(map[uuid.UUID]model.User),
		UserGroups:            make(map[uuid.UUID]model.UserGroup),
		UserGroupMembers:      make(map[uuid.UUID]map[uuid.UUID]bool),
		Tags:                  make(map[uuid.UUID]model.Tag),
		UserTags:              make(map[uuid.UUID]map[uuid.UUID]model.UsersTag),
		Channels:              make(map[uuid.UUID]model.Channel),
		ChannelSubscribes:     make(map[uuid.UUID]map[uuid.UUID]bool),
		PrivateChannelMembers: make(map[uuid.UUID]map[uuid.UUID]bool),
		Messages:              make(map[uuid.UUID]model.Message),
		MessageUnreads:        make(map[uuid.UUID]map[uuid.UUID]bool),
		MessageReports:        make([]model.MessageReport, 0),
		Pins:                  make(map[uuid.UUID]model.Pin),
		Stars:                 make(map[uuid.UUID]map[uuid.UUID]bool),
		Mute:                  make(map[uuid.UUID]map[uuid.UUID]bool),
		Stamps:                make(map[uuid.UUID]model.Stamp),
		Files:                 make(map[uuid.UUID]model.File),
		FilesACL:              make(map[uuid.UUID]map[uuid.UUID]bool),
	}
	_, _ = r.CreateUser("traq", "traq", role.Admin)
	return r
}

func (r *TestRepository) Sync() (bool, error) {
	panic("implement me")
}

func (r *TestRepository) GetFS() storage.FileStorage {
	return r.FS
}

func (r *TestRepository) CreateUser(name, password string, role gorbac.Role) (*model.User, error) {
	r.UsersLock.Lock()
	defer r.UsersLock.Unlock()

	for _, v := range r.Users {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}

	salt := utils.GenerateSalt()
	user := model.User{
		ID:        uuid.NewV4(),
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

	iconID, err := r.GenerateIconFile(user.Name)
	if err != nil {
		return nil, err
	}
	user.Icon = iconID

	r.Users[user.ID] = user
	return &user, nil
}

func (r *TestRepository) GetUser(id uuid.UUID) (*model.User, error) {
	r.UsersLock.RLock()
	u, ok := r.Users[id]
	r.UsersLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (r *TestRepository) GetUserByName(name string) (*model.User, error) {
	r.UsersLock.RLock()
	defer r.UsersLock.RUnlock()
	for _, u := range r.Users {
		u := u
		if u.Name == name {
			return &u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *TestRepository) GetUsers() ([]*model.User, error) {
	r.UsersLock.RLock()
	result := make([]*model.User, 0, len(r.Users))
	for _, u := range r.Users {
		u := u
		result = append(result, &u)
	}
	r.UsersLock.RUnlock()
	return result, nil
}

func (r *TestRepository) UserExists(id uuid.UUID) (bool, error) {
	r.UsersLock.RLock()
	_, ok := r.Users[id]
	r.UsersLock.RUnlock()
	return ok, nil
}

func (r *TestRepository) ChangeUserDisplayName(id uuid.UUID, displayName string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if utf8.RuneCountInString(displayName) > 64 {
		return errors.New("displayName must be <=64 characters")
	}
	r.UsersLock.Lock()
	u, ok := r.Users[id]
	if ok {
		u.DisplayName = displayName
		u.UpdatedAt = time.Now()
		r.Users[id] = u
	}
	r.UsersLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeUserPassword(id uuid.UUID, password string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	salt := utils.GenerateSalt()
	hashed := utils.HashPassword(password, salt)
	r.UsersLock.Lock()
	u, ok := r.Users[id]
	if ok {
		u.Salt = hex.EncodeToString(salt)
		u.Password = hex.EncodeToString(hashed)
		u.UpdatedAt = time.Now()
		r.Users[id] = u
	}
	r.UsersLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeUserIcon(id, fileID uuid.UUID) error {
	if id == uuid.Nil || fileID == uuid.Nil {
		return repository.ErrNilID
	}
	r.UsersLock.Lock()
	u, ok := r.Users[id]
	if ok {
		u.Icon = fileID
		u.UpdatedAt = time.Now()
		r.Users[id] = u
	}
	r.UsersLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeUserTwitterID(id uuid.UUID, twitterID string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	if err := validator.ValidateVar(twitterID, "twitterid"); err != nil {
		return err
	}
	r.UsersLock.Lock()
	u, ok := r.Users[id]
	if ok {
		u.TwitterID = twitterID
		u.UpdatedAt = time.Now()
		r.Users[id] = u
	}
	r.UsersLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeUserAccountStatus(id uuid.UUID, status model.UserAccountStatus) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.UsersLock.Lock()
	defer r.UsersLock.Unlock()
	u, ok := r.Users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.Status = status
	u.UpdatedAt = time.Now()
	r.Users[id] = u
	return nil
}

func (r *TestRepository) UpdateUserLastOnline(id uuid.UUID, t time.Time) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.UsersLock.Lock()
	u, ok := r.Users[id]
	if ok {
		u.LastOnline = &t
		u.UpdatedAt = time.Now()
		r.Users[id] = u
	}
	r.UsersLock.Unlock()
	return nil
}

func (r *TestRepository) IsUserOnline(id uuid.UUID) bool {
	return false
}

func (r *TestRepository) GetUserLastOnline(id uuid.UUID) (time.Time, error) {
	r.UsersLock.RLock()
	u, ok := r.Users[id]
	r.UsersLock.RUnlock()
	if !ok {
		return time.Time{}, repository.ErrNotFound
	}
	if u.LastOnline != nil {
		return *u.LastOnline, nil
	}
	return time.Time{}, nil
}

func (r *TestRepository) GetHeartbeatStatus(channelID uuid.UUID) (model.HeartbeatStatus, bool) {
	panic("implement me")
}

func (r *TestRepository) UpdateHeartbeatStatus(userID, channelID uuid.UUID, status string) {
	panic("implement me")
}

func (r *TestRepository) CreateUserGroup(name, description string, adminID uuid.UUID) (*model.UserGroup, error) {
	g := model.UserGroup{
		ID:          uuid.NewV4(),
		Name:        name,
		Description: description,
		AdminUserID: adminID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}

	r.UserGroupsLock.Lock()
	defer r.UserGroupsLock.Unlock()
	for _, v := range r.UserGroups {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	r.UserGroups[g.ID] = g
	return &g, nil
}

func (r *TestRepository) UpdateUserGroup(id uuid.UUID, args repository.UpdateUserGroupNameArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	r.UserGroupsLock.Lock()
	defer r.UserGroupsLock.Unlock()
	g, ok := r.UserGroups[id]
	if !ok {
		return repository.ErrNotFound
	}
	if len(args.Name) > 0 {
		for _, v := range r.UserGroups {
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
	r.UserGroups[id] = g
	return nil
}

func (r *TestRepository) DeleteUserGroup(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.UserGroupsLock.Lock()
	defer r.UserGroupsLock.Unlock()
	r.UserGroupMembersLock.Lock()
	defer r.UserGroupMembersLock.Unlock()
	if _, ok := r.UserGroups[id]; !ok {
		return repository.ErrNotFound
	}
	delete(r.UserGroups, id)
	delete(r.UserGroupMembers, id)
	return nil
}

func (r *TestRepository) GetUserGroup(id uuid.UUID) (*model.UserGroup, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	r.UserGroupsLock.RLock()
	g, ok := r.UserGroups[id]
	r.UserGroupsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &g, nil
}

func (r *TestRepository) GetUserGroupByName(name string) (*model.UserGroup, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	r.UserGroupsLock.RLock()
	defer r.UserGroupsLock.RUnlock()
	for _, v := range r.UserGroups {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *TestRepository) GetUserBelongingGroupIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	groups := make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return groups, nil
	}
	r.UserGroupMembersLock.RLock()
	for gid, users := range r.UserGroupMembers {
		for uid := range users {
			if uid == userID {
				groups = append(groups, gid)
				break
			}
		}
	}
	r.UserGroupMembersLock.RUnlock()
	return groups, nil
}

func (r *TestRepository) GetAllUserGroups() ([]*model.UserGroup, error) {
	groups := make([]*model.UserGroup, 0)
	r.UserGroupsLock.RLock()
	for _, v := range r.UserGroups {
		v := v
		groups = append(groups, &v)
	}
	r.UserGroupsLock.RUnlock()
	return groups, nil
}

func (r *TestRepository) AddUserToGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	r.UserGroupMembersLock.Lock()
	users, ok := r.UserGroupMembers[groupID]
	if !ok {
		users = make(map[uuid.UUID]bool)
		r.UserGroupMembers[groupID] = users
	}
	users[userID] = true
	r.UserGroupMembersLock.Unlock()
	return nil
}

func (r *TestRepository) RemoveUserFromGroup(userID, groupID uuid.UUID) error {
	if userID == uuid.Nil || groupID == uuid.Nil {
		return repository.ErrNilID
	}
	r.UserGroupMembersLock.Lock()
	users, ok := r.UserGroupMembers[groupID]
	if ok {
		delete(users, userID)
	}
	r.UserGroupMembersLock.Unlock()
	return nil
}

func (r *TestRepository) GetUserGroupMemberIDs(groupID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	if groupID == uuid.Nil {
		return ids, repository.ErrNotFound
	}
	r.UserGroupsLock.RLock()
	_, ok := r.UserGroups[groupID]
	r.UserGroupsLock.RUnlock()
	if !ok {
		return ids, repository.ErrNotFound
	}
	r.UserGroupMembersLock.RLock()
	for uid := range r.UserGroupMembers[groupID] {
		ids = append(ids, uid)
	}
	r.UserGroupMembersLock.RUnlock()
	return ids, nil
}

func (r *TestRepository) CreateTag(name string, restricted bool, tagType string) (*model.Tag, error) {
	r.TagsLock.Lock()
	defer r.TagsLock.Unlock()
	for _, t := range r.Tags {
		if t.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	t := model.Tag{
		ID:         uuid.NewV4(),
		Name:       name,
		Restricted: restricted,
		Type:       tagType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	r.Tags[t.ID] = t
	return &t, nil
}

func (r *TestRepository) ChangeTagType(id uuid.UUID, tagType string) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.TagsLock.Lock()
	t, ok := r.Tags[id]
	if ok {
		t.Type = tagType
		t.UpdatedAt = time.Now()
		r.Tags[id] = t
	}
	r.TagsLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeTagRestrict(id uuid.UUID, restrict bool) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.TagsLock.Lock()
	t, ok := r.Tags[id]
	if ok {
		t.Restricted = restrict
		t.UpdatedAt = time.Now()
		r.Tags[id] = t
	}
	r.TagsLock.Unlock()
	return nil
}

func (r *TestRepository) GetAllTags() ([]*model.Tag, error) {
	result := make([]*model.Tag, 0)
	r.TagsLock.RLock()
	for _, v := range r.Tags {
		v := v
		result = append(result, &v)
	}
	r.TagsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetTagByID(id uuid.UUID) (*model.Tag, error) {
	r.TagsLock.RLock()
	t, ok := r.Tags[id]
	r.TagsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &t, nil
}

func (r *TestRepository) GetTagByName(name string) (*model.Tag, error) {
	r.TagsLock.RLock()
	defer r.TagsLock.RUnlock()
	for _, t := range r.Tags {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *TestRepository) GetOrCreateTagByName(name string) (*model.Tag, error) {
	if len(name) == 0 {
		return nil, repository.ErrNotFound
	}
	r.TagsLock.Lock()
	defer r.TagsLock.Unlock()
	for _, t := range r.Tags {
		if t.Name == name {
			return &t, nil
		}
	}
	t := model.Tag{
		ID:        uuid.NewV4(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.Tags[t.ID] = t
	return &t, nil
}

func (r *TestRepository) AddUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	ut := model.UsersTag{
		UserID:    userID,
		TagID:     tagID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.UserTagsLock.Lock()
	tags, ok := r.UserTags[userID]
	if !ok {
		tags = make(map[uuid.UUID]model.UsersTag)
		r.UserTags[userID] = tags
	}
	if _, ok := tags[tagID]; ok {
		return repository.ErrAlreadyExists
	}
	tags[tagID] = ut
	r.UserTagsLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	r.UserTagsLock.Lock()
	defer r.UserTagsLock.Unlock()
	for id, tag := range r.UserTags[userID] {
		if id == tagID {
			tag.IsLocked = locked
			tag.UpdatedAt = time.Now()
			r.UserTags[userID][tagID] = tag
			return nil
		}
	}
	return nil
}

func (r *TestRepository) DeleteUserTag(userID, tagID uuid.UUID) error {
	if userID == uuid.Nil || tagID == uuid.Nil {
		return repository.ErrNilID
	}
	r.UserTagsLock.Lock()
	tags, ok := r.UserTags[userID]
	if ok {
		delete(tags, tagID)
	}
	r.UserTagsLock.Unlock()
	return nil
}

func (r *TestRepository) GetUserTag(userID, tagID uuid.UUID) (*model.UsersTag, error) {
	r.UserTagsLock.RLock()
	defer r.UserTagsLock.RUnlock()
	tags, ok := r.UserTags[userID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	ut, ok := tags[tagID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	r.TagsLock.RLock()
	ut.Tag = r.Tags[ut.TagID]
	r.TagsLock.RUnlock()
	return &ut, nil
}

func (r *TestRepository) GetUserTagsByUserID(userID uuid.UUID) ([]*model.UsersTag, error) {
	tags := make([]*model.UsersTag, 0)
	r.UserTagsLock.RLock()
	for tid, ut := range r.UserTags[userID] {
		ut := ut
		r.TagsLock.RLock()
		ut.Tag = r.Tags[tid]
		r.TagsLock.RUnlock()
		tags = append(tags, &ut)
	}
	r.UserTagsLock.RUnlock()
	return tags, nil
}

func (r *TestRepository) GetUsersByTag(tag string) ([]*model.User, error) {
	users := make([]*model.User, 0)
	r.TagsLock.RLock()
	tid := uuid.Nil
	for _, t := range r.Tags {
		if t.Name == tag {
			tid = t.ID
		}
	}
	r.TagsLock.RUnlock()
	if tid == uuid.Nil {
		return users, nil
	}
	r.UserTagsLock.RLock()
	for uid, tags := range r.UserTags {
		if _, ok := tags[tid]; ok {
			r.UsersLock.RLock()
			u, ok := r.Users[uid]
			r.UsersLock.RUnlock()
			if ok {
				users = append(users, &u)
			}
		}
	}
	r.UserTagsLock.RUnlock()
	return users, nil
}

func (r *TestRepository) GetUserIDsByTag(tag string) ([]uuid.UUID, error) {
	users := make([]uuid.UUID, 0)
	r.TagsLock.RLock()
	tid := uuid.Nil
	for _, t := range r.Tags {
		if t.Name == tag {
			tid = t.ID
		}
	}
	r.TagsLock.RUnlock()
	if tid == uuid.Nil {
		return users, nil
	}
	r.UserTagsLock.RLock()
	for uid, tags := range r.UserTags {
		if _, ok := tags[tid]; ok {
			users = append(users, uid)
		}
	}
	r.UserTagsLock.RUnlock()
	return users, nil
}

func (r *TestRepository) GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error) {
	users := make([]uuid.UUID, 0)
	r.UserTagsLock.RLock()
	for uid, tags := range r.UserTags {
		if _, ok := tags[tagID]; ok {
			users = append(users, uid)
		}
	}
	r.UserTagsLock.RUnlock()
	return users, nil
}

func (r *TestRepository) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := r.IsChannelPresent(name, parent); err != nil {
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
		pCh, err := r.GetChannel(parent)
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

			parent, err = r.GetChannel(parent.ParentID)
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
		ID:        uuid.NewV4(),
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
	r.ChannelsLock.Lock()
	r.Channels[ch.ID] = ch
	r.ChannelsLock.Unlock()
	return &ch, nil
}

func (r *TestRepository) CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error) {
	validMember := make([]uuid.UUID, 0, len(members))
	for _, v := range members {
		ok, err := r.UserExists(v)
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
	if has, err := r.IsChannelPresent(name, uuid.Nil); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	ch := model.Channel{
		ID:        uuid.NewV4(),
		Name:      name,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsPublic:  false,
		IsForced:  false,
		IsVisible: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.ChannelsLock.Lock()
	r.Channels[ch.ID] = ch
	for _, v := range validMember {
		_ = r.AddPrivateChannelMember(ch.ID, v)
	}
	r.ChannelsLock.Unlock()
	return &ch, nil
}

func (r *TestRepository) CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error) {
	// ダイレクトメッセージルートの子チャンネルは作れない
	if parentID == dmChannelRootUUID {
		return nil, repository.ErrForbidden
	}

	// 親チャンネル検証
	pCh, err := r.GetChannel(parentID)
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
	if has, err := r.IsChannelPresent(name, pCh.ID); err != nil {
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

		parent, err = r.GetChannel(parent.ParentID)
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
		ID:        uuid.NewV4(),
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
		r.ChannelsLock.Lock()
		r.Channels[ch.ID] = ch
		r.ChannelsLock.Unlock()
	} else {
		// 非公開チャンネル
		ch.IsPublic = false

		// 親チャンネルとメンバーは同じ
		ids, err := r.GetPrivateChannelMemberIDs(pCh.ID)
		if err != nil {
			return nil, err
		}

		r.ChannelsLock.Lock()
		r.Channels[ch.ID] = ch
		for _, v := range ids {
			_ = r.AddPrivateChannelMember(ch.ID, v)
		}
		r.ChannelsLock.Unlock()
	}
	return &ch, nil
}

func (r *TestRepository) UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.ChannelsLock.Lock()
	ch, ok := r.Channels[channelID]
	if ok {
		if visibility != nil {
			ch.IsVisible = *visibility
		}
		if forced != nil {
			ch.IsForced = *forced
		}
		ch.UpdatedAt = time.Now()
		r.Channels[channelID] = ch
	}
	r.ChannelsLock.Unlock()
	return nil
}

func (r *TestRepository) UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.ChannelsLock.Lock()
	ch, ok := r.Channels[channelID]
	if ok {
		ch.Topic = topic
		ch.UpdatedAt = time.Now()
		r.Channels[channelID] = ch
	}
	r.ChannelsLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeChannelName(channelID uuid.UUID, name string) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return err
	}

	// チャンネル取得
	ch, err := r.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの名前は変更不可能
	if ch.IsDMChannel() {
		return repository.ErrForbidden
	}

	// チャンネル名重複を確認
	if has, err := r.IsChannelPresent(name, ch.ParentID); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
	}

	// 更新
	r.ChannelsLock.Lock()
	nch, ok := r.Channels[channelID]
	if ok {
		nch.Name = name
		nch.UpdatedAt = time.Now()
		r.Channels[channelID] = nch
	}
	r.ChannelsLock.Unlock()
	return nil
}

func (r *TestRepository) ChangeChannelParent(channelID, parent uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル取得
	ch, err := r.GetChannel(channelID)
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
		pCh, err := r.GetChannel(parent)
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

			pCh, err = r.GetChannel(pCh.ParentID)
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
		bottom, err := r.GetChannelDepth(ch.ID) // 子孫 (自分を含む)
		if err != nil {
			return err
		}
		depth += bottom
		if depth > model.MaxChannelDepth {
			return repository.ErrChannelDepthLimitation
		}
	}

	// チャンネル名検証
	if has, err := r.IsChannelPresent(ch.Name, parent); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
	}

	// 更新
	r.ChannelsLock.Lock()
	nch, ok := r.Channels[channelID]
	if ok {
		nch.ParentID = parent
		nch.UpdatedAt = time.Now()
		r.Channels[channelID] = nch
	}
	r.ChannelsLock.Unlock()
	return nil
}

func (r *TestRepository) DeleteChannel(channelID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	desc, err := r.GetDescendantChannelIDs(channelID)
	if err != nil {
		return err
	}
	r.ChannelsLock.Lock()
	for _, id := range append(desc, channelID) {
		delete(r.Channels, id)
	}
	r.ChannelsLock.Unlock()
	return nil
}

func (r *TestRepository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	r.ChannelsLock.RLock()
	ch, ok := r.Channels[channelID]
	r.ChannelsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &ch, nil
}

func (r *TestRepository) GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error) {
	r.MessagesLock.RLock()
	m, ok := r.Messages[messageID]
	r.MessagesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	r.ChannelsLock.RLock()
	ch, ok := r.Channels[m.ChannelID]
	r.ChannelsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &ch, nil
}

func (r *TestRepository) GetChannelsByUserID(userID uuid.UUID) ([]*model.Channel, error) {
	result := make([]*model.Channel, 0)
	r.ChannelsLock.RLock()
	for _, ch := range r.Channels {
		ch := ch
		if ch.IsPublic {
			result = append(result, &ch)
		} else if userID != uuid.Nil {
			ok, _ := r.IsUserPrivateChannelMember(ch.ID, userID)
			if ok {
				result = append(result, &ch)
			}
		}
	}
	r.ChannelsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	panic("implement me")
}

func (r *TestRepository) GetAllChannels() ([]*model.Channel, error) {
	r.ChannelsLock.RLock()
	result := make([]*model.Channel, 0, len(r.Channels))
	for _, c := range r.Channels {
		c := c
		result = append(result, &c)
	}
	r.ChannelsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) IsChannelPresent(name string, parent uuid.UUID) (bool, error) {
	r.ChannelsLock.RLock()
	defer r.ChannelsLock.RUnlock()
	for _, ch := range r.Channels {
		if ch.Name == name && ch.ParentID == parent {
			return true, nil
		}
	}
	return false, nil
}

func (r *TestRepository) IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	r.ChannelsLock.RLock()
	ch, ok := r.Channels[channelID]
	r.ChannelsLock.RUnlock()
	if !ok {
		return false, nil
	}
	if ch.IsPublic {
		return true, nil
	}
	return r.IsUserPrivateChannelMember(channelID, userID)
}

func (r *TestRepository) GetParentChannel(channelID uuid.UUID) (*model.Channel, error) {
	r.ChannelsLock.RLock()
	defer r.ChannelsLock.RUnlock()
	ch, ok := r.Channels[channelID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if ch.ParentID == uuid.Nil {
		return nil, nil
	}
	pCh, ok := r.Channels[ch.ParentID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &pCh, nil
}

func (r *TestRepository) GetChildrenChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0)
	r.ChannelsLock.RLock()
	for cid, ch := range r.Channels {
		if ch.ParentID == channelID {
			result = append(result, cid)
		}
	}
	r.ChannelsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetDescendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var descendants []uuid.UUID
	children, err := r.GetChildrenChannelIDs(channelID)
	if err != nil {
		return nil, err
	}
	descendants = append(descendants, children...)
	for _, v := range children {
		sub, err := r.GetDescendantChannelIDs(v)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, sub...)
	}
	return descendants, nil
}

func (r *TestRepository) GetAscendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var ascendants []uuid.UUID
	parent, err := r.GetParentChannel(channelID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	} else if parent == nil {
		return []uuid.UUID{}, nil
	}
	ascendants = append(ascendants, parent.ID)
	sub, err := r.GetAscendantChannelIDs(parent.ID)
	if err != nil {
		return nil, err
	}
	ascendants = append(ascendants, sub...)
	return ascendants, nil
}

func (r *TestRepository) GetChannelPath(id uuid.UUID) (string, error) {
	panic("implement me")
}

func (r *TestRepository) GetChannelDepth(id uuid.UUID) (int, error) {
	children, err := r.GetChildrenChannelIDs(id)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, v := range children {
		d, err := r.GetChannelDepth(v)
		if err != nil {
			return 0, err
		}
		if max < d {
			max = d
		}
	}
	return max + 1, nil
}

func (r *TestRepository) AddPrivateChannelMember(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	r.PrivateChannelMembersLock.Lock()
	uids, ok := r.PrivateChannelMembers[channelID]
	if !ok {
		uids = make(map[uuid.UUID]bool)
	}
	uids[userID] = true
	r.PrivateChannelMembers[channelID] = uids
	r.PrivateChannelMembersLock.Unlock()
	return nil
}

func (r *TestRepository) GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0)
	r.PrivateChannelMembersLock.RLock()
	for uid := range r.PrivateChannelMembers[channelID] {
		result = append(result, uid)
	}
	r.PrivateChannelMembersLock.RUnlock()
	return result, nil
}

func (r *TestRepository) IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error) {
	r.PrivateChannelMembersLock.RLock()
	defer r.PrivateChannelMembersLock.RUnlock()
	uids, ok := r.PrivateChannelMembers[channelID]
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

func (r *TestRepository) SubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.ChannelSubscribesLock.Lock()
	chMap, ok := r.ChannelSubscribes[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	r.ChannelSubscribes[userID] = chMap
	r.ChannelSubscribesLock.Unlock()
	return nil
}

func (r *TestRepository) UnsubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.ChannelSubscribesLock.Lock()
	chMap, ok := r.ChannelSubscribes[userID]
	if ok {
		delete(chMap, channelID)
		r.ChannelSubscribes[userID] = chMap
	}
	r.ChannelSubscribesLock.Unlock()
	return nil
}

func (r *TestRepository) GetSubscribingUserIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	r.ChannelSubscribesLock.RLock()
	result := make([]uuid.UUID, 0)
	for uid, chMap := range r.ChannelSubscribes {
		for cid := range chMap {
			if cid == channelID {
				result = append(result, uid)
			}
		}
	}
	r.ChannelSubscribesLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetSubscribedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	r.ChannelSubscribesLock.RLock()
	result := make([]uuid.UUID, 0)
	chMap, ok := r.ChannelSubscribes[userID]
	if ok {
		for id := range chMap {
			result = append(result, id)
		}
	}
	r.ChannelSubscribesLock.RUnlock()
	return result, nil
}

func (r *TestRepository) CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	m := &model.Message{
		ID:        uuid.NewV4(),
		UserID:    userID,
		ChannelID: channelID,
		Text:      text,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	r.MessagesLock.Lock()
	r.Messages[m.ID] = *m
	r.MessagesLock.Unlock()
	return m, nil
}

func (r *TestRepository) UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	if len(text) == 0 {
		return errors.New("text is empty")
	}

	r.MessagesLock.Lock()
	defer r.MessagesLock.Unlock()
	m, ok := r.Messages[messageID]
	if !ok {
		return repository.ErrNotFound
	}
	m.Text = text
	m.UpdatedAt = time.Now()
	r.Messages[messageID] = m
	return nil
}

func (r *TestRepository) DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}

	r.MessagesLock.Lock()
	defer r.MessagesLock.Unlock()
	if _, ok := r.Messages[messageID]; !ok {
		return repository.ErrNotFound
	}
	delete(r.Messages, messageID)
	return nil
}

func (r *TestRepository) GetMessageByID(messageID uuid.UUID) (*model.Message, error) {
	r.MessagesLock.RLock()
	m, ok := r.Messages[messageID]
	r.MessagesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &m, nil
}

func (r *TestRepository) GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) ([]*model.Message, error) {
	tmp := make([]*model.Message, 0)
	r.MessagesLock.RLock()
	for _, v := range r.Messages {
		if v.ChannelID == channelID {
			v := v
			tmp = append(tmp, &v)
		}
	}
	r.MessagesLock.RUnlock()
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

func (r *TestRepository) SetMessageUnread(userID, messageID uuid.UUID) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return repository.ErrNilID
	}
	r.MessageUnreadsLock.Lock()
	mMap, ok := r.MessageUnreads[userID]
	if !ok {
		mMap = make(map[uuid.UUID]bool)
	}
	mMap[messageID] = true
	r.MessageUnreads[userID] = mMap
	r.MessageUnreadsLock.Unlock()
	return nil
}

func (r *TestRepository) GetUnreadMessagesByUserID(userID uuid.UUID) ([]*model.Message, error) {
	result := make([]*model.Message, 0)
	r.MessageUnreadsLock.RLock()
	r.MessagesLock.RLock()
	for uid, mMap := range r.MessageUnreads {
		if uid != userID {
			continue
		}
		for mid := range mMap {
			m, ok := r.Messages[mid]
			if ok {
				result = append(result, &m)
			}
		}
	}
	r.MessagesLock.RUnlock()
	r.MessageUnreadsLock.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		return result[j].CreatedAt.After(result[i].CreatedAt)
	})
	return result, nil
}

func (r *TestRepository) DeleteUnreadsByMessageID(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	r.MessageUnreadsLock.Lock()
	for _, mMap := range r.MessageUnreads {
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
	r.MessageUnreadsLock.Unlock()
	return nil
}

func (r *TestRepository) DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	r.MessageUnreadsLock.Lock()
	r.MessagesLock.RLock()
	for uid, mMap := range r.MessageUnreads {
		if uid != userID {
			continue
		}
		var deleted []uuid.UUID
		for mid := range mMap {
			m, ok := r.Messages[mid]
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
	r.MessagesLock.RUnlock()
	r.MessageUnreadsLock.Unlock()
	return nil
}

func (r *TestRepository) GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error) {
	panic("implement me")
}

func (r *TestRepository) CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	if messageID == uuid.Nil || reporterID == uuid.Nil {
		return repository.ErrNilID
	}

	// make report
	report := model.MessageReport{
		ID:        uuid.NewV4(),
		MessageID: messageID,
		Reporter:  reporterID,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	r.MessageReportsLock.Lock()
	defer r.MessageReportsLock.Unlock()
	for _, v := range r.MessageReports {
		if v.MessageID == messageID && v.Reporter == reporterID {
			return repository.ErrAlreadyExists
		}
	}
	r.MessageReports = append(r.MessageReports, report)
	return nil
}

func (r *TestRepository) GetMessageReports(offset, limit int) ([]*model.MessageReport, error) {
	r.MessageReportsLock.RLock()
	l := len(r.MessageReports)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = math.MaxInt32
	}
	result := make([]*model.MessageReport, 0)
	for i := offset; i < l && i < offset+limit; i++ {
		re := r.MessageReports[i]
		result = append(result, &re)
	}
	r.MessageReportsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetMessageReportsByMessageID(messageID uuid.UUID) ([]*model.MessageReport, error) {
	r.MessageReportsLock.RLock()
	result := make([]*model.MessageReport, 0)
	for _, v := range r.MessageReports {
		if v.MessageID == messageID {
			v := v
			result = append(result, &v)
		}
	}
	r.MessageReportsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) GetMessageReportsByReporterID(reporterID uuid.UUID) ([]*model.MessageReport, error) {
	r.MessageReportsLock.RLock()
	result := make([]*model.MessageReport, 0)
	for _, v := range r.MessageReports {
		if v.Reporter == reporterID {
			v := v
			result = append(result, &v)
		}
	}
	r.MessageReportsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) AddStampToMessage(messageID, stampID, userID uuid.UUID) (ms *model.MessageStamp, err error) {
	panic("implement me")
}

func (r *TestRepository) RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error) {
	panic("implement me")
}

func (r *TestRepository) GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error) {
	return []*model.MessageStamp{}, nil
}

func (r *TestRepository) GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error) {
	panic("implement me")
}

func (r *TestRepository) CreateStamp(name string, fileID, userID uuid.UUID) (s *model.Stamp, err error) {
	if fileID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	stamp := &model.Stamp{
		ID:        uuid.NewV4(),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
	}
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	r.StampsLock.Lock()
	defer r.StampsLock.Unlock()
	for _, v := range r.Stamps {
		if v.Name == name {
			return nil, repository.ErrAlreadyExists
		}
	}
	r.Stamps[stamp.ID] = *stamp
	return stamp, nil
}

func (r *TestRepository) UpdateStamp(id uuid.UUID, name string, fileID uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}

	r.StampsLock.Lock()
	defer r.StampsLock.Unlock()
	s, ok := r.Stamps[id]

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
	r.Stamps[id] = s
	return nil
}

func (r *TestRepository) GetStamp(id uuid.UUID) (*model.Stamp, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	r.StampsLock.RLock()
	s, ok := r.Stamps[id]
	r.StampsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &s, nil
}

func (r *TestRepository) DeleteStamp(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.StampsLock.Lock()
	defer r.StampsLock.Unlock()
	if _, ok := r.Stamps[id]; !ok {
		return repository.ErrNotFound
	}
	delete(r.Stamps, id)
	return nil
}

func (r *TestRepository) GetAllStamps() (stamps []*model.Stamp, err error) {
	r.StampsLock.RLock()
	for _, v := range r.Stamps {
		v := v
		stamps = append(stamps, &v)
	}
	r.StampsLock.RUnlock()
	return
}

func (r *TestRepository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	r.StampsLock.RLock()
	_, ok := r.Stamps[id]
	r.StampsLock.RUnlock()
	return ok, nil
}

func (r *TestRepository) IsStampNameDuplicate(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	r.StampsLock.RUnlock()
	defer r.StampsLock.RUnlock()
	for _, v := range r.Stamps {
		if v.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *TestRepository) GetClipFolder(id uuid.UUID) (*model.ClipFolder, error) {
	panic("implement me")
}

func (r *TestRepository) GetClipFolders(userID uuid.UUID) ([]*model.ClipFolder, error) {
	panic("implement me")
}

func (r *TestRepository) CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error) {
	panic("implement me")
}

func (r *TestRepository) UpdateClipFolderName(id uuid.UUID, name string) error {
	panic("implement me")
}

func (r *TestRepository) DeleteClipFolder(id uuid.UUID) error {
	panic("implement me")
}

func (r *TestRepository) GetClipMessage(id uuid.UUID) (*model.Clip, error) {
	panic("implement me")
}

func (r *TestRepository) GetClipMessages(folderID uuid.UUID) ([]*model.Clip, error) {
	panic("implement me")
}

func (r *TestRepository) GetClipMessagesByUser(userID uuid.UUID) ([]*model.Clip, error) {
	panic("implement me")
}

func (r *TestRepository) CreateClip(messageID, folderID, userID uuid.UUID) (*model.Clip, error) {
	panic("implement me")
}

func (r *TestRepository) ChangeClipFolder(clipID, folderID uuid.UUID) error {
	panic("implement me")
}

func (r *TestRepository) DeleteClip(id uuid.UUID) error {
	panic("implement me")
}

func (r *TestRepository) MuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.MuteLock.Lock()
	chMap, ok := r.Mute[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	r.Mute[userID] = chMap
	r.MuteLock.Unlock()
	return nil
}

func (r *TestRepository) UnmuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.MuteLock.Lock()
	chMap, ok := r.Mute[userID]
	if ok {
		delete(chMap, channelID)
		r.Stars[userID] = chMap
	}
	r.MuteLock.Unlock()
	return nil
}

func (r *TestRepository) GetMutedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	r.MuteLock.RLock()
	chMap, ok := r.Mute[userID]
	if ok {
		for id := range chMap {
			ids = append(ids, id)
		}
	}
	r.MuteLock.RUnlock()
	return ids, nil
}

func (r *TestRepository) GetMuteUserIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	r.MuteLock.RLock()
	for uid, chMap := range r.Mute {
		if chMap[channelID] {
			ids = append(ids, uid)
		}
	}
	r.MuteLock.RUnlock()
	return ids, nil
}

func (r *TestRepository) IsChannelMuted(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	r.MuteLock.RLock()
	defer r.MuteLock.RUnlock()
	chMap, ok := r.Mute[userID]
	if !ok {
		return false, nil
	}
	return chMap[channelID], nil
}

func (r *TestRepository) AddStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.StarsLock.Lock()
	chMap, ok := r.Stars[userID]
	if !ok {
		chMap = make(map[uuid.UUID]bool)
	}
	chMap[channelID] = true
	r.Stars[userID] = chMap
	r.StarsLock.Unlock()
	return nil
}

func (r *TestRepository) RemoveStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	r.StarsLock.Lock()
	chMap, ok := r.Stars[userID]
	if ok {
		delete(chMap, channelID)
		r.Stars[userID] = chMap
	}
	r.StarsLock.Unlock()
	return nil
}

func (r *TestRepository) GetStaredChannels(userID uuid.UUID) ([]uuid.UUID, error) {
	r.StarsLock.RLock()
	result := make([]uuid.UUID, 0)
	chMap, ok := r.Stars[userID]
	if ok {
		for id := range chMap {
			result = append(result, id)
		}
	}
	r.StarsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return uuid.Nil, repository.ErrNilID
	}
	r.PinsLock.Lock()
	defer r.PinsLock.Unlock()
	for _, pin := range r.Pins {
		if pin.MessageID == messageID {
			return pin.ID, nil
		}
	}
	p := model.Pin{
		ID:        uuid.NewV4(),
		MessageID: messageID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	r.Pins[p.ID] = p
	return p.ID, nil
}

func (r *TestRepository) GetPin(id uuid.UUID) (*model.Pin, error) {
	r.PinsLock.RLock()
	pin, ok := r.Pins[id]
	r.PinsLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	r.MessagesLock.RLock()
	pin.Message = r.Messages[pin.MessageID]
	r.MessagesLock.RUnlock()
	return &pin, nil
}

func (r *TestRepository) IsPinned(messageID uuid.UUID) (bool, error) {
	r.PinsLock.RLock()
	defer r.PinsLock.RUnlock()
	for _, p := range r.Pins {
		if p.MessageID == messageID {
			return true, nil
		}
	}
	return false, nil
}

func (r *TestRepository) DeletePin(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	r.PinsLock.Lock()
	delete(r.Pins, id)
	r.PinsLock.Unlock()
	return nil
}

func (r *TestRepository) GetPinsByChannelID(channelID uuid.UUID) ([]*model.Pin, error) {
	result := make([]*model.Pin, 0)
	r.PinsLock.RLock()
	r.MessagesLock.RLock()
	for _, p := range r.Pins {
		m, ok := r.Messages[p.MessageID]
		if ok && m.ChannelID == channelID {
			p := p
			p.Message = m
			result = append(result, &p)
		}
	}
	r.MessagesLock.RUnlock()
	r.PinsLock.RUnlock()
	return result, nil
}

func (r *TestRepository) RegisterDevice(userID uuid.UUID, token string) (*model.Device, error) {
	panic("implement me")
}

func (r *TestRepository) UnregisterDevice(token string) (err error) {
	panic("implement me")
}

func (r *TestRepository) GetDevicesByUserID(user uuid.UUID) (result []*model.Device, err error) {
	panic("implement me")
}

func (r *TestRepository) GetDeviceTokensByUserID(user uuid.UUID) (result []string, err error) {
	panic("implement me")
}

func (r *TestRepository) GetAllDevices() (result []*model.Device, err error) {
	panic("implement me")
}

func (r *TestRepository) GetAllDeviceTokens() (result []string, err error) {
	panic("implement me")
}

func (r *TestRepository) OpenFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := r.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	rc, err := r.FS.OpenFileByKey(meta.GetKey())
	return meta, rc, err
}

func (r *TestRepository) OpenThumbnailFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error) {
	meta, err := r.GetFileMeta(fileID)
	if err != nil {
		return nil, nil, err
	}
	if meta.HasThumbnail {
		rc, err := r.FS.OpenFileByKey(meta.GetThumbKey())
		return meta, rc, err
	}
	return meta, nil, repository.ErrNotFound
}

func (r *TestRepository) GetFileMeta(fileID uuid.UUID) (*model.File, error) {
	if fileID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	r.FilesLock.RLock()
	meta, ok := r.Files[fileID]
	r.FilesLock.RUnlock()
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &meta, nil
}

func (r *TestRepository) DeleteFile(fileID uuid.UUID) error {
	if fileID == uuid.Nil {
		return repository.ErrNilID
	}
	r.FilesLock.Lock()
	defer r.FilesLock.Unlock()
	meta, ok := r.Files[fileID]
	if !ok {
		return repository.ErrNotFound
	}
	delete(r.Files, fileID)
	return r.FS.DeleteByKey(meta.GetKey())
}

func (r *TestRepository) GenerateIconFile(salt string) (uuid.UUID, error) {
	img, _ := thumb.EncodeToPNG(utils.GenerateIcon(salt))
	file, e := r.SaveFile(fmt.Sprintf("%s.png", salt), img, int64(img.Len()), "image/png", model.FileTypeIcon, uuid.Nil)
	return file.ID, e
}

func (r *TestRepository) SaveFile(name string, src io.Reader, size int64, mimeType string, fType string, creatorID uuid.UUID) (*model.File, error) {
	return r.SaveFileWithACL(name, src, size, mimeType, fType, creatorID, repository.ACL{uuid.Nil: true})
}

func (r *TestRepository) SaveFileWithACL(name string, src io.Reader, size int64, mimeType string, fType string, creatorID uuid.UUID, read repository.ACL) (*model.File, error) {
	f := &model.File{
		ID:        uuid.NewV4(),
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
		if err := r.FS.SaveByKey(fileSrc, f.GetKey(), f.Name, f.Mime, f.Type); err != nil {
			return err
		}
		return nil
	})

	// サムネイルの生成
	eg.Go(func() error {
		// アップロードされたファイルの拡張子が間違えてたり、変なの送ってきた場合
		// サムネイルを生成しないだけで全体のエラーにはしない
		defer thumbSrc.Close()
		size, _ := r.generateThumbnail(ctx, f, thumbSrc)
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
	r.FilesLock.Lock()
	r.FilesACLLock.Lock()
	r.Files[f.ID] = *f
	r.FilesACL[f.ID] = read
	r.FilesACLLock.Unlock()
	r.FilesLock.Unlock()
	return f, nil
}

func (r *TestRepository) RegenerateThumbnail(fileID uuid.UUID) (bool, error) {
	return false, nil
}

func (r *TestRepository) IsFileAccessible(fileID, userID uuid.UUID) (bool, error) {
	if fileID == uuid.Nil {
		return false, repository.ErrNilID
	}
	r.FilesLock.RLock()
	_, ok := r.Files[fileID]
	r.FilesLock.RUnlock()
	if !ok {
		return false, repository.ErrNotFound
	}

	var allow bool
	r.FilesACLLock.RLock()
	defer r.FilesACLLock.RUnlock()
	for uid, a := range r.FilesACL[fileID] {
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

func (r *TestRepository) generateThumbnail(ctx context.Context, f *model.File, src io.Reader) (image.Rectangle, error) {
	img, err := thumb.Generate(ctx, src, f.Mime)
	if err != nil {
		return image.ZR, err
	}
	b, _ := thumb.EncodeToPNG(img)
	if err := r.FS.SaveByKey(b, f.GetThumbKey(), f.GetThumbKey()+".png", "image/png", model.FileTypeThumbnail); err != nil {
		return image.ZR, err
	}
	return img.Bounds(), nil
}

func (r *TestRepository) CreateWebhook(name, description string, channelID, creatorID, iconFileID uuid.UUID) (model.Webhook, error) {
	panic("implement me")
}

func (r *TestRepository) UpdateWebhook(id uuid.UUID, name, description *string, channelID uuid.UUID) error {
	panic("implement me")
}

func (r *TestRepository) DeleteWebhook(id uuid.UUID) error {
	panic("implement me")
}

func (r *TestRepository) GetWebhook(id uuid.UUID) (model.Webhook, error) {
	panic("implement me")
}

func (r *TestRepository) GetAllWebhooks() ([]model.Webhook, error) {
	panic("implement me")
}

func (r *TestRepository) GetWebhooksByCreator(creatorID uuid.UUID) ([]model.Webhook, error) {
	panic("implement me")
}
