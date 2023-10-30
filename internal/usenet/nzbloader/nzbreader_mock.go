// Code generated by MockGen. DO NOT EDIT.
// Source: ./nzbreader.go

// Package nzbloader is a generated GoMock package.
package nzbloader

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	usenet "github.com/javi11/usenet-drive/internal/usenet"
	nzb "github.com/javi11/usenet-drive/pkg/nzb"
)

// MockNzbReader is a mock of NzbReader interface.
type MockNzbReader struct {
	ctrl     *gomock.Controller
	recorder *MockNzbReaderMockRecorder
}

// MockNzbReaderMockRecorder is the mock recorder for MockNzbReader.
type MockNzbReaderMockRecorder struct {
	mock *MockNzbReader
}

// NewMockNzbReader creates a new mock instance.
func NewMockNzbReader(ctrl *gomock.Controller) *MockNzbReader {
	mock := &MockNzbReader{ctrl: ctrl}
	mock.recorder = &MockNzbReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNzbReader) EXPECT() *MockNzbReaderMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockNzbReader) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockNzbReaderMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockNzbReader)(nil).Close))
}

// GetGroups mocks base method.
func (m *MockNzbReader) GetGroups() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroups")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGroups indicates an expected call of GetGroups.
func (mr *MockNzbReaderMockRecorder) GetGroups() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroups", reflect.TypeOf((*MockNzbReader)(nil).GetGroups))
}

// GetMetadata mocks base method.
func (m *MockNzbReader) GetMetadata() (usenet.Metadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMetadata")
	ret0, _ := ret[0].(usenet.Metadata)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMetadata indicates an expected call of GetMetadata.
func (mr *MockNzbReaderMockRecorder) GetMetadata() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMetadata", reflect.TypeOf((*MockNzbReader)(nil).GetMetadata))
}

// GetSegment mocks base method.
func (m *MockNzbReader) GetSegment(segmentIndex int) (*nzb.NzbSegment, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSegment", segmentIndex)
	ret0, _ := ret[0].(*nzb.NzbSegment)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetSegment indicates an expected call of GetSegment.
func (mr *MockNzbReaderMockRecorder) GetSegment(segmentIndex interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSegment", reflect.TypeOf((*MockNzbReader)(nil).GetSegment), segmentIndex)
}
