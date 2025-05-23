// Code generated by MockGen. DO NOT EDIT.
// Source: channel.go

// Package mock_repository is a generated GoMock package.
package mock_repository

import (
	reflect "reflect"
	time "time"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	model "github.com/traPtitech/traQ/model"
	repository "github.com/traPtitech/traQ/repository"
	set "github.com/traPtitech/traQ/utils/set"
)

// MockChannelRepository is a mock of ChannelRepository interface.
type MockChannelRepository struct {
	ctrl     *gomock.Controller
	recorder *MockChannelRepositoryMockRecorder
}

// MockChannelRepositoryMockRecorder is the mock recorder for MockChannelRepository.
type MockChannelRepositoryMockRecorder struct {
	mock *MockChannelRepository
}

// NewMockChannelRepository creates a new mock instance.
func NewMockChannelRepository(ctrl *gomock.Controller) *MockChannelRepository {
	mock := &MockChannelRepository{ctrl: ctrl}
	mock.recorder = &MockChannelRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChannelRepository) EXPECT() *MockChannelRepositoryMockRecorder {
	return m.recorder
}

// ArchiveChannels mocks base method.
func (m *MockChannelRepository) ArchiveChannels(ids []uuid.UUID) ([]*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ArchiveChannels", ids)
	ret0, _ := ret[0].([]*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ArchiveChannels indicates an expected call of ArchiveChannels.
func (mr *MockChannelRepositoryMockRecorder) ArchiveChannels(ids interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ArchiveChannels", reflect.TypeOf((*MockChannelRepository)(nil).ArchiveChannels), ids)
}

// ChangeChannelSubscription mocks base method.
func (m *MockChannelRepository) ChangeChannelSubscription(channelID uuid.UUID, args repository.ChangeChannelSubscriptionArgs) ([]uuid.UUID, []uuid.UUID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeChannelSubscription", channelID, args)
	ret0, _ := ret[0].([]uuid.UUID)
	ret1, _ := ret[1].([]uuid.UUID)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ChangeChannelSubscription indicates an expected call of ChangeChannelSubscription.
func (mr *MockChannelRepositoryMockRecorder) ChangeChannelSubscription(channelID, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeChannelSubscription", reflect.TypeOf((*MockChannelRepository)(nil).ChangeChannelSubscription), channelID, args)
}

// CreateChannel mocks base method.
func (m *MockChannelRepository) CreateChannel(ch model.Channel, privateMembers set.UUID, dm bool) (*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateChannel", ch, privateMembers, dm)
	ret0, _ := ret[0].(*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateChannel indicates an expected call of CreateChannel.
func (mr *MockChannelRepositoryMockRecorder) CreateChannel(ch, privateMembers, dm interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateChannel", reflect.TypeOf((*MockChannelRepository)(nil).CreateChannel), ch, privateMembers, dm)
}

// GetChannel mocks base method.
func (m *MockChannelRepository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChannel", channelID)
	ret0, _ := ret[0].(*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChannel indicates an expected call of GetChannel.
func (mr *MockChannelRepositoryMockRecorder) GetChannel(channelID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChannel", reflect.TypeOf((*MockChannelRepository)(nil).GetChannel), channelID)
}

// GetChannelEvents mocks base method.
func (m *MockChannelRepository) GetChannelEvents(query repository.ChannelEventsQuery) ([]*model.ChannelEvent, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChannelEvents", query)
	ret0, _ := ret[0].([]*model.ChannelEvent)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetChannelEvents indicates an expected call of GetChannelEvents.
func (mr *MockChannelRepositoryMockRecorder) GetChannelEvents(query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChannelEvents", reflect.TypeOf((*MockChannelRepository)(nil).GetChannelEvents), query)
}

// GetChannelStats mocks base method.
func (m *MockChannelRepository) GetChannelStats(channelID uuid.UUID, excludeDeletedMessages bool) (*repository.ChannelStats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChannelStats", channelID, excludeDeletedMessages)
	ret0, _ := ret[0].(*repository.ChannelStats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChannelStats indicates an expected call of GetChannelStats.
func (mr *MockChannelRepositoryMockRecorder) GetChannelStats(channelID, excludeDeletedMessages interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChannelStats", reflect.TypeOf((*MockChannelRepository)(nil).GetChannelStats), channelID, excludeDeletedMessages)
}

// GetChannelSubscriptions mocks base method.
func (m *MockChannelRepository) GetChannelSubscriptions(query repository.ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChannelSubscriptions", query)
	ret0, _ := ret[0].([]*model.UserSubscribeChannel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChannelSubscriptions indicates an expected call of GetChannelSubscriptions.
func (mr *MockChannelRepositoryMockRecorder) GetChannelSubscriptions(query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChannelSubscriptions", reflect.TypeOf((*MockChannelRepository)(nil).GetChannelSubscriptions), query)
}

// GetDirectMessageChannel mocks base method.
func (m *MockChannelRepository) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDirectMessageChannel", user1, user2)
	ret0, _ := ret[0].(*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDirectMessageChannel indicates an expected call of GetDirectMessageChannel.
func (mr *MockChannelRepositoryMockRecorder) GetDirectMessageChannel(user1, user2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDirectMessageChannel", reflect.TypeOf((*MockChannelRepository)(nil).GetDirectMessageChannel), user1, user2)
}

// GetDirectMessageChannelMapping mocks base method.
func (m *MockChannelRepository) GetDirectMessageChannelMapping(userID uuid.UUID) ([]*model.DMChannelMapping, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDirectMessageChannelMapping", userID)
	ret0, _ := ret[0].([]*model.DMChannelMapping)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDirectMessageChannelMapping indicates an expected call of GetDirectMessageChannelMapping.
func (mr *MockChannelRepositoryMockRecorder) GetDirectMessageChannelMapping(userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDirectMessageChannelMapping", reflect.TypeOf((*MockChannelRepository)(nil).GetDirectMessageChannelMapping), userID)
}

// GetPrivateChannelMemberIDs mocks base method.
func (m *MockChannelRepository) GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrivateChannelMemberIDs", channelID)
	ret0, _ := ret[0].([]uuid.UUID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPrivateChannelMemberIDs indicates an expected call of GetPrivateChannelMemberIDs.
func (mr *MockChannelRepositoryMockRecorder) GetPrivateChannelMemberIDs(channelID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrivateChannelMemberIDs", reflect.TypeOf((*MockChannelRepository)(nil).GetPrivateChannelMemberIDs), channelID)
}

// GetPublicChannels mocks base method.
func (m *MockChannelRepository) GetPublicChannels() ([]*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicChannels")
	ret0, _ := ret[0].([]*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPublicChannels indicates an expected call of GetPublicChannels.
func (mr *MockChannelRepositoryMockRecorder) GetPublicChannels() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicChannels", reflect.TypeOf((*MockChannelRepository)(nil).GetPublicChannels))
}

// RecordChannelEvent mocks base method.
func (m *MockChannelRepository) RecordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordChannelEvent", channelID, eventType, detail, datetime)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecordChannelEvent indicates an expected call of RecordChannelEvent.
func (mr *MockChannelRepositoryMockRecorder) RecordChannelEvent(channelID, eventType, detail, datetime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordChannelEvent", reflect.TypeOf((*MockChannelRepository)(nil).RecordChannelEvent), channelID, eventType, detail, datetime)
}

// UpdateChannel mocks base method.
func (m *MockChannelRepository) UpdateChannel(channelID uuid.UUID, args repository.UpdateChannelArgs) (*model.Channel, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateChannel", channelID, args)
	ret0, _ := ret[0].(*model.Channel)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateChannel indicates an expected call of UpdateChannel.
func (mr *MockChannelRepositoryMockRecorder) UpdateChannel(channelID, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateChannel", reflect.TypeOf((*MockChannelRepository)(nil).UpdateChannel), channelID, args)
}
