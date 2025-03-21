package devicetwin

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/mocks/beehive"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
)

const (
	//TestModule is name of test.
	TestModule = "test"
	//DeviceTwinModuleName is name of twin
	DeviceTwinModuleName = "twin"
)

func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	add := &common.ModuleInfo{
		ModuleName: TestModule,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(add)
	beehiveContext.AddModuleGroup(TestModule, TestModule)
}

// TestName is function to test Name().
func TestName(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name string
		want string
	}{
		{
			name: "DeviceTwinNametest",
			want: "twin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := &DeviceTwin{}
			assert.Equal(tt.want, dt.Name(), "DeviceTwin.Name() = %v, want %v", dt.Name(), tt.want)
		})
	}
}

// TestGroup is function to test Group().
func TestGroup(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name string
		want string
	}{
		{
			name: "DeviceTwinGroupTest",
			want: "twin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := &DeviceTwin{}
			assert.Equal(tt.want, dt.Group(), "DeviceTwin.Group() = %v, want %v", dt.Group(), tt.want)
		})
	}
}

// TestStart is function to test Start().
func TestStart(t *testing.T) {
	assert := assert.New(t)

	//test is for sending test messages from devicetwin module.
	var test model.Message
	// ormerMock is mocked Ormer implementation.
	var ormerMock *beego.MockOrmer
	// querySeterMock is mocked QuerySeter implementation.
	var querySeterMock *beego.MockQuerySeter
	// fakeModule is mocked implementation of TestModule.
	var fakeModule *beehive.MockModule

	const delay = 10 * time.Millisecond
	const maxRetries = 5
	retry := 0

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	fakeModule = beehive.NewMockModule(mockCtrl)
	dbm.DBAccess = ormerMock

	fakeModule.EXPECT().Enable().Return(true).Times(1)
	fakeModule.EXPECT().Name().Return(TestModule).MaxTimes(5)
	fakeModule.EXPECT().Group().Return(TestModule).MaxTimes(5)

	core.Register(fakeModule)
	add := &common.ModuleInfo{
		ModuleName: TestModule,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(add)
	dt := newDeviceTwin(true)
	core.Register(dt)
	addDt := &common.ModuleInfo{
		ModuleName: dt.Name(),
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(addDt)
	beehiveContext.AddModuleGroup(dt.Name(), dt.Group())
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).MinTimes(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).MaxTimes(5)
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), nil).MinTimes(1)
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), nil).MaxTimes(5)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).MinTimes(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).MaxTimes(5)
	go dt.Start()
	time.Sleep(delay)
	retry++
	// Sending a message from devicetwin module to the created fake module(TestModule) to check context is initialized properly.
	beehiveContext.Send(TestModule, test)
	_, err := beehiveContext.Receive(TestModule)
	assert.NoError(err)
	//Checking whether Mem,Twin,Device and Comm modules are registered and started successfully.
	tests := []struct {
		name       string
		moduleName string
	}{
		{
			name:       "MemModuleHealthCheck",
			moduleName: dtcommon.MemModule,
		},
		{
			name:       "TwinModuleHealthCheck",
			moduleName: dtcommon.TwinModule,
		},
		{
			name:       "DeviceModuleHealthCheck",
			moduleName: dtcommon.DeviceModule,
		},
		{
			name:       "CommModuleHealthCheck",
			moduleName: dtcommon.CommModule,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			moduleCheck := false
			for retry < maxRetries {
				for _, module := range dt.DTModules {
					if test.moduleName == module.Name {
						moduleCheck = true
						err := dt.DTContexts.HeartBeat(test.moduleName, "ping")
						assert.NoError(err, "Heartbeat of module %v is expired and dtcontroller will start it again", test.moduleName)
						break
					}
				}
				if moduleCheck {
					break
				}
				time.Sleep(delay)
				retry++
			}
			assert.Less(retry, maxRetries, "Registration of module %v failed", test.moduleName)
		})
	}
}
