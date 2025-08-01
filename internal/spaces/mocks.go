// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/digitalocean/godo (interfaces: SpacesKeysService,CDNService)
//
// Generated by this command:
//
//	mockgen -destination=./mocks.go -package spaces github.com/digitalocean/godo SpacesKeysService,CDNService
//

// Package spaces is a generated GoMock package.
package spaces

import (
	context "context"
	reflect "reflect"

	godo "github.com/digitalocean/godo"
	gomock "go.uber.org/mock/gomock"
)

// MockSpacesKeysService is a mock of SpacesKeysService interface.
type MockSpacesKeysService struct {
	ctrl     *gomock.Controller
	recorder *MockSpacesKeysServiceMockRecorder
}

// MockSpacesKeysServiceMockRecorder is the mock recorder for MockSpacesKeysService.
type MockSpacesKeysServiceMockRecorder struct {
	mock *MockSpacesKeysService
}

// NewMockSpacesKeysService creates a new mock instance.
func NewMockSpacesKeysService(ctrl *gomock.Controller) *MockSpacesKeysService {
	mock := &MockSpacesKeysService{ctrl: ctrl}
	mock.recorder = &MockSpacesKeysServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSpacesKeysService) EXPECT() *MockSpacesKeysServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockSpacesKeysService) Create(arg0 context.Context, arg1 *godo.SpacesKeyCreateRequest) (*godo.SpacesKey, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(*godo.SpacesKey)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Create indicates an expected call of Create.
func (mr *MockSpacesKeysServiceMockRecorder) Create(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockSpacesKeysService)(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockSpacesKeysService) Delete(arg0 context.Context, arg1 string) (*godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(*godo.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockSpacesKeysServiceMockRecorder) Delete(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockSpacesKeysService)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *MockSpacesKeysService) Get(arg0 context.Context, arg1 string) (*godo.SpacesKey, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*godo.SpacesKey)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockSpacesKeysServiceMockRecorder) Get(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockSpacesKeysService)(nil).Get), arg0, arg1)
}

// List mocks base method.
func (m *MockSpacesKeysService) List(arg0 context.Context, arg1 *godo.ListOptions) ([]*godo.SpacesKey, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]*godo.SpacesKey)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// List indicates an expected call of List.
func (mr *MockSpacesKeysServiceMockRecorder) List(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockSpacesKeysService)(nil).List), arg0, arg1)
}

// Update mocks base method.
func (m *MockSpacesKeysService) Update(arg0 context.Context, arg1 string, arg2 *godo.SpacesKeyUpdateRequest) (*godo.SpacesKey, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1, arg2)
	ret0, _ := ret[0].(*godo.SpacesKey)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Update indicates an expected call of Update.
func (mr *MockSpacesKeysServiceMockRecorder) Update(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockSpacesKeysService)(nil).Update), arg0, arg1, arg2)
}

// MockCDNService is a mock of CDNService interface.
type MockCDNService struct {
	ctrl     *gomock.Controller
	recorder *MockCDNServiceMockRecorder
}

// MockCDNServiceMockRecorder is the mock recorder for MockCDNService.
type MockCDNServiceMockRecorder struct {
	mock *MockCDNService
}

// NewMockCDNService creates a new mock instance.
func NewMockCDNService(ctrl *gomock.Controller) *MockCDNService {
	mock := &MockCDNService{ctrl: ctrl}
	mock.recorder = &MockCDNServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCDNService) EXPECT() *MockCDNServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockCDNService) Create(arg0 context.Context, arg1 *godo.CDNCreateRequest) (*godo.CDN, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(*godo.CDN)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Create indicates an expected call of Create.
func (mr *MockCDNServiceMockRecorder) Create(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockCDNService)(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockCDNService) Delete(arg0 context.Context, arg1 string) (*godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(*godo.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockCDNServiceMockRecorder) Delete(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockCDNService)(nil).Delete), arg0, arg1)
}

// FlushCache mocks base method.
func (m *MockCDNService) FlushCache(arg0 context.Context, arg1 string, arg2 *godo.CDNFlushCacheRequest) (*godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlushCache", arg0, arg1, arg2)
	ret0, _ := ret[0].(*godo.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FlushCache indicates an expected call of FlushCache.
func (mr *MockCDNServiceMockRecorder) FlushCache(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlushCache", reflect.TypeOf((*MockCDNService)(nil).FlushCache), arg0, arg1, arg2)
}

// Get mocks base method.
func (m *MockCDNService) Get(arg0 context.Context, arg1 string) (*godo.CDN, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*godo.CDN)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockCDNServiceMockRecorder) Get(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCDNService)(nil).Get), arg0, arg1)
}

// List mocks base method.
func (m *MockCDNService) List(arg0 context.Context, arg1 *godo.ListOptions) ([]godo.CDN, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", arg0, arg1)
	ret0, _ := ret[0].([]godo.CDN)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// List indicates an expected call of List.
func (mr *MockCDNServiceMockRecorder) List(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockCDNService)(nil).List), arg0, arg1)
}

// UpdateCustomDomain mocks base method.
func (m *MockCDNService) UpdateCustomDomain(arg0 context.Context, arg1 string, arg2 *godo.CDNUpdateCustomDomainRequest) (*godo.CDN, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCustomDomain", arg0, arg1, arg2)
	ret0, _ := ret[0].(*godo.CDN)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// UpdateCustomDomain indicates an expected call of UpdateCustomDomain.
func (mr *MockCDNServiceMockRecorder) UpdateCustomDomain(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCustomDomain", reflect.TypeOf((*MockCDNService)(nil).UpdateCustomDomain), arg0, arg1, arg2)
}

// UpdateTTL mocks base method.
func (m *MockCDNService) UpdateTTL(arg0 context.Context, arg1 string, arg2 *godo.CDNUpdateTTLRequest) (*godo.CDN, *godo.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTTL", arg0, arg1, arg2)
	ret0, _ := ret[0].(*godo.CDN)
	ret1, _ := ret[1].(*godo.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// UpdateTTL indicates an expected call of UpdateTTL.
func (mr *MockCDNServiceMockRecorder) UpdateTTL(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTTL", reflect.TypeOf((*MockCDNService)(nil).UpdateTTL), arg0, arg1, arg2)
}
