// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler (interfaces: UserService)
//
// Generated by this command:
//
//	mockgen -destination=../service/mocks/mock_user.go -package=mockservice github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler UserService
//

// Package mockservice is a generated GoMock package.
package mockservice

import (
	context "context"
	reflect "reflect"

	dto "github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	model "github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	gomock "go.uber.org/mock/gomock"
)

// MockUserService is a mock of UserService interface.
type MockUserService struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceMockRecorder
}

// MockUserServiceMockRecorder is the mock recorder for MockUserService.
type MockUserServiceMockRecorder struct {
	mock *MockUserService
}

// NewMockUserService creates a new mock instance.
func NewMockUserService(ctrl *gomock.Controller) *MockUserService {
	mock := &MockUserService{ctrl: ctrl}
	mock.recorder = &MockUserServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserService) EXPECT() *MockUserServiceMockRecorder {
	return m.recorder
}

// CreateAndInvite mocks base method.
func (m *MockUserService) CreateAndInvite(arg0 context.Context, arg1 dto.CreateUserReq) (*model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAndInvite", arg0, arg1)
	ret0, _ := ret[0].(*model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAndInvite indicates an expected call of CreateAndInvite.
func (mr *MockUserServiceMockRecorder) CreateAndInvite(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAndInvite", reflect.TypeOf((*MockUserService)(nil).CreateAndInvite), arg0, arg1)
}

// GetByID mocks base method.
func (m *MockUserService) GetByID(arg0 context.Context, arg1 int64) (*model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", arg0, arg1)
	ret0, _ := ret[0].(*model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockUserServiceMockRecorder) GetByID(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockUserService)(nil).GetByID), arg0, arg1)
}

// Update mocks base method.
func (m *MockUserService) Update(arg0 context.Context, arg1 dto.UpdateUserReq) (*model.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(*model.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockUserServiceMockRecorder) Update(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockUserService)(nil).Update), arg0, arg1)
}
