// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openshift/assisted-service/internal/versions (interfaces: OSImages)

// Package versions is a generated GoMock package.
package versions

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/openshift/assisted-service/models"
)

// MockOSImages is a mock of OSImages interface.
type MockOSImages struct {
	ctrl     *gomock.Controller
	recorder *MockOSImagesMockRecorder
}

// MockOSImagesMockRecorder is the mock recorder for MockOSImages.
type MockOSImagesMockRecorder struct {
	mock *MockOSImages
}

// NewMockOSImages creates a new mock instance.
func NewMockOSImages(ctrl *gomock.Controller) *MockOSImages {
	mock := &MockOSImages{ctrl: ctrl}
	mock.recorder = &MockOSImagesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOSImages) EXPECT() *MockOSImagesMockRecorder {
	return m.recorder
}

// GetCPUArchitectures mocks base method.
func (m *MockOSImages) GetCPUArchitectures(arg0 string) []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCPUArchitectures", arg0)
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetCPUArchitectures indicates an expected call of GetCPUArchitectures.
func (mr *MockOSImagesMockRecorder) GetCPUArchitectures(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCPUArchitectures", reflect.TypeOf((*MockOSImages)(nil).GetCPUArchitectures), arg0)
}

// GetLatestOsImage mocks base method.
func (m *MockOSImages) GetLatestOsImage(arg0 string) (*models.OsImage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestOsImage", arg0)
	ret0, _ := ret[0].(*models.OsImage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestOsImage indicates an expected call of GetLatestOsImage.
func (mr *MockOSImagesMockRecorder) GetLatestOsImage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestOsImage", reflect.TypeOf((*MockOSImages)(nil).GetLatestOsImage), arg0)
}

// GetOpenshiftVersions mocks base method.
func (m *MockOSImages) GetOpenshiftVersions() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOpenshiftVersions")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetOpenshiftVersions indicates an expected call of GetOpenshiftVersions.
func (mr *MockOSImagesMockRecorder) GetOpenshiftVersions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOpenshiftVersions", reflect.TypeOf((*MockOSImages)(nil).GetOpenshiftVersions))
}

// GetOsImage mocks base method.
func (m *MockOSImages) GetOsImage(arg0, arg1 string) (*models.OsImage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOsImage", arg0, arg1)
	ret0, _ := ret[0].(*models.OsImage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOsImage indicates an expected call of GetOsImage.
func (mr *MockOSImagesMockRecorder) GetOsImage(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOsImage", reflect.TypeOf((*MockOSImages)(nil).GetOsImage), arg0, arg1)
}

// GetOsImageOrLatest mocks base method.
func (m *MockOSImages) GetOsImageOrLatest(arg0, arg1 string) (*models.OsImage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOsImageOrLatest", arg0, arg1)
	ret0, _ := ret[0].(*models.OsImage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOsImageOrLatest indicates an expected call of GetOsImageOrLatest.
func (mr *MockOSImagesMockRecorder) GetOsImageOrLatest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOsImageOrLatest", reflect.TypeOf((*MockOSImages)(nil).GetOsImageOrLatest), arg0, arg1)
}