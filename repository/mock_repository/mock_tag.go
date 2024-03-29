// Code generated by MockGen. DO NOT EDIT.
// Source: tag.go

// Package mock_repository is a generated GoMock package.
package mock_repository

import (
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	model "github.com/traPtitech/traQ/model"
)

// MockTagRepository is a mock of TagRepository interface.
type MockTagRepository struct {
	ctrl     *gomock.Controller
	recorder *MockTagRepositoryMockRecorder
}

// MockTagRepositoryMockRecorder is the mock recorder for MockTagRepository.
type MockTagRepositoryMockRecorder struct {
	mock *MockTagRepository
}

// NewMockTagRepository creates a new mock instance.
func NewMockTagRepository(ctrl *gomock.Controller) *MockTagRepository {
	mock := &MockTagRepository{ctrl: ctrl}
	mock.recorder = &MockTagRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTagRepository) EXPECT() *MockTagRepositoryMockRecorder {
	return m.recorder
}

// AddUserTag mocks base method.
func (m *MockTagRepository) AddUserTag(userID, tagID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddUserTag", userID, tagID)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddUserTag indicates an expected call of AddUserTag.
func (mr *MockTagRepositoryMockRecorder) AddUserTag(userID, tagID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddUserTag", reflect.TypeOf((*MockTagRepository)(nil).AddUserTag), userID, tagID)
}

// ChangeUserTagLock mocks base method.
func (m *MockTagRepository) ChangeUserTagLock(userID, tagID uuid.UUID, locked bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeUserTagLock", userID, tagID, locked)
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeUserTagLock indicates an expected call of ChangeUserTagLock.
func (mr *MockTagRepositoryMockRecorder) ChangeUserTagLock(userID, tagID, locked interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeUserTagLock", reflect.TypeOf((*MockTagRepository)(nil).ChangeUserTagLock), userID, tagID, locked)
}

// DeleteUserTag mocks base method.
func (m *MockTagRepository) DeleteUserTag(userID, tagID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUserTag", userID, tagID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUserTag indicates an expected call of DeleteUserTag.
func (mr *MockTagRepositoryMockRecorder) DeleteUserTag(userID, tagID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUserTag", reflect.TypeOf((*MockTagRepository)(nil).DeleteUserTag), userID, tagID)
}

// GetOrCreateTag mocks base method.
func (m *MockTagRepository) GetOrCreateTag(name string) (*model.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrCreateTag", name)
	ret0, _ := ret[0].(*model.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrCreateTag indicates an expected call of GetOrCreateTag.
func (mr *MockTagRepositoryMockRecorder) GetOrCreateTag(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrCreateTag", reflect.TypeOf((*MockTagRepository)(nil).GetOrCreateTag), name)
}

// GetTagByID mocks base method.
func (m *MockTagRepository) GetTagByID(id uuid.UUID) (*model.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTagByID", id)
	ret0, _ := ret[0].(*model.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTagByID indicates an expected call of GetTagByID.
func (mr *MockTagRepositoryMockRecorder) GetTagByID(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTagByID", reflect.TypeOf((*MockTagRepository)(nil).GetTagByID), id)
}

// GetUserIDsByTagID mocks base method.
func (m *MockTagRepository) GetUserIDsByTagID(tagID uuid.UUID) ([]uuid.UUID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserIDsByTagID", tagID)
	ret0, _ := ret[0].([]uuid.UUID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserIDsByTagID indicates an expected call of GetUserIDsByTagID.
func (mr *MockTagRepositoryMockRecorder) GetUserIDsByTagID(tagID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserIDsByTagID", reflect.TypeOf((*MockTagRepository)(nil).GetUserIDsByTagID), tagID)
}

// GetUserTag mocks base method.
func (m *MockTagRepository) GetUserTag(userID, tagID uuid.UUID) (model.UserTag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserTag", userID, tagID)
	ret0, _ := ret[0].(model.UserTag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserTag indicates an expected call of GetUserTag.
func (mr *MockTagRepositoryMockRecorder) GetUserTag(userID, tagID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserTag", reflect.TypeOf((*MockTagRepository)(nil).GetUserTag), userID, tagID)
}

// GetUserTagsByUserID mocks base method.
func (m *MockTagRepository) GetUserTagsByUserID(userID uuid.UUID) ([]model.UserTag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserTagsByUserID", userID)
	ret0, _ := ret[0].([]model.UserTag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserTagsByUserID indicates an expected call of GetUserTagsByUserID.
func (mr *MockTagRepositoryMockRecorder) GetUserTagsByUserID(userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserTagsByUserID", reflect.TypeOf((*MockTagRepository)(nil).GetUserTagsByUserID), userID)
}
