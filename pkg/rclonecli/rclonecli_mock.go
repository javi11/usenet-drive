// Code generated by MockGen. DO NOT EDIT.
// Source: ./rclonecli.go

// Package rclonecli is a generated GoMock package.
package rclonecli

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockRcloneRcClient is a mock of RcloneRcClient interface.
type MockRcloneRcClient struct {
	ctrl     *gomock.Controller
	recorder *MockRcloneRcClientMockRecorder
}

// MockRcloneRcClientMockRecorder is the mock recorder for MockRcloneRcClient.
type MockRcloneRcClientMockRecorder struct {
	mock *MockRcloneRcClient
}

// NewMockRcloneRcClient creates a new mock instance.
func NewMockRcloneRcClient(ctrl *gomock.Controller) *MockRcloneRcClient {
	mock := &MockRcloneRcClient{ctrl: ctrl}
	mock.recorder = &MockRcloneRcClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRcloneRcClient) EXPECT() *MockRcloneRcClientMockRecorder {
	return m.recorder
}

// RefreshCache mocks base method.
func (m *MockRcloneRcClient) RefreshCache(ctx context.Context, dir string, async, recursive bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshCache", ctx, dir, async, recursive)
	ret0, _ := ret[0].(error)
	return ret0
}

// RefreshCache indicates an expected call of RefreshCache.
func (mr *MockRcloneRcClientMockRecorder) RefreshCache(ctx, dir, async, recursive interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshCache", reflect.TypeOf((*MockRcloneRcClient)(nil).RefreshCache), ctx, dir, async, recursive)
}
