// Code generated by MockGen. DO NOT EDIT.
// Source: ../../go-choria/server/agents/agents.go

// Package provision is a generated GoMock package.
package provision

import (
	context "context"
	json "encoding/json"
	aagent "github.com/choria-io/go-choria/aagent"
	choria "github.com/choria-io/go-choria/choria"
	lifecycle "github.com/choria-io/go-choria/lifecycle"
	protocol "github.com/choria-io/go-choria/protocol"
	agents "github.com/choria-io/go-choria/server/agents"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockAgent is a mock of Agent interface
type MockAgent struct {
	ctrl     *gomock.Controller
	recorder *MockAgentMockRecorder
}

// MockAgentMockRecorder is the mock recorder for MockAgent
type MockAgentMockRecorder struct {
	mock *MockAgent
}

// NewMockAgent creates a new mock instance
func NewMockAgent(ctrl *gomock.Controller) *MockAgent {
	mock := &MockAgent{ctrl: ctrl}
	mock.recorder = &MockAgentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAgent) EXPECT() *MockAgentMockRecorder {
	return m.recorder
}

// Metadata mocks base method
func (m *MockAgent) Metadata() *agents.Metadata {
	ret := m.ctrl.Call(m, "Metadata")
	ret0, _ := ret[0].(*agents.Metadata)
	return ret0
}

// Metadata indicates an expected call of Metadata
func (mr *MockAgentMockRecorder) Metadata() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Metadata", reflect.TypeOf((*MockAgent)(nil).Metadata))
}

// Name mocks base method
func (m *MockAgent) Name() string {
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockAgentMockRecorder) Name() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockAgent)(nil).Name))
}

// HandleMessage mocks base method
func (m *MockAgent) HandleMessage(arg0 context.Context, arg1 *choria.Message, arg2 protocol.Request, arg3 choria.ConnectorInfo, arg4 chan *agents.AgentReply) {
	m.ctrl.Call(m, "HandleMessage", arg0, arg1, arg2, arg3, arg4)
}

// HandleMessage indicates an expected call of HandleMessage
func (mr *MockAgentMockRecorder) HandleMessage(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleMessage", reflect.TypeOf((*MockAgent)(nil).HandleMessage), arg0, arg1, arg2, arg3, arg4)
}

// SetServerInfo mocks base method
func (m *MockAgent) SetServerInfo(arg0 agents.ServerInfoSource) {
	m.ctrl.Call(m, "SetServerInfo", arg0)
}

// SetServerInfo indicates an expected call of SetServerInfo
func (mr *MockAgentMockRecorder) SetServerInfo(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetServerInfo", reflect.TypeOf((*MockAgent)(nil).SetServerInfo), arg0)
}

// ServerInfo mocks base method
func (m *MockAgent) ServerInfo() agents.ServerInfoSource {
	ret := m.ctrl.Call(m, "ServerInfo")
	ret0, _ := ret[0].(agents.ServerInfoSource)
	return ret0
}

// ServerInfo indicates an expected call of ServerInfo
func (mr *MockAgentMockRecorder) ServerInfo() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerInfo", reflect.TypeOf((*MockAgent)(nil).ServerInfo))
}

// ShouldActivate mocks base method
func (m *MockAgent) ShouldActivate() bool {
	ret := m.ctrl.Call(m, "ShouldActivate")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldActivate indicates an expected call of ShouldActivate
func (mr *MockAgentMockRecorder) ShouldActivate() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldActivate", reflect.TypeOf((*MockAgent)(nil).ShouldActivate))
}

// MockServerInfoSource is a mock of ServerInfoSource interface
type MockServerInfoSource struct {
	ctrl     *gomock.Controller
	recorder *MockServerInfoSourceMockRecorder
}

// MockServerInfoSourceMockRecorder is the mock recorder for MockServerInfoSource
type MockServerInfoSourceMockRecorder struct {
	mock *MockServerInfoSource
}

// NewMockServerInfoSource creates a new mock instance
func NewMockServerInfoSource(ctrl *gomock.Controller) *MockServerInfoSource {
	mock := &MockServerInfoSource{ctrl: ctrl}
	mock.recorder = &MockServerInfoSourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockServerInfoSource) EXPECT() *MockServerInfoSourceMockRecorder {
	return m.recorder
}

// AgentMetadata mocks base method
func (m *MockServerInfoSource) AgentMetadata(arg0 string) (agents.Metadata, bool) {
	ret := m.ctrl.Call(m, "AgentMetadata", arg0)
	ret0, _ := ret[0].(agents.Metadata)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// AgentMetadata indicates an expected call of AgentMetadata
func (mr *MockServerInfoSourceMockRecorder) AgentMetadata(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AgentMetadata", reflect.TypeOf((*MockServerInfoSource)(nil).AgentMetadata), arg0)
}

// Classes mocks base method
func (m *MockServerInfoSource) Classes() []string {
	ret := m.ctrl.Call(m, "Classes")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Classes indicates an expected call of Classes
func (mr *MockServerInfoSourceMockRecorder) Classes() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Classes", reflect.TypeOf((*MockServerInfoSource)(nil).Classes))
}

// ConfigFile mocks base method
func (m *MockServerInfoSource) ConfigFile() string {
	ret := m.ctrl.Call(m, "ConfigFile")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConfigFile indicates an expected call of ConfigFile
func (mr *MockServerInfoSourceMockRecorder) ConfigFile() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigFile", reflect.TypeOf((*MockServerInfoSource)(nil).ConfigFile))
}

// ConnectedServer mocks base method
func (m *MockServerInfoSource) ConnectedServer() string {
	ret := m.ctrl.Call(m, "ConnectedServer")
	ret0, _ := ret[0].(string)
	return ret0
}

// ConnectedServer indicates an expected call of ConnectedServer
func (mr *MockServerInfoSourceMockRecorder) ConnectedServer() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedServer", reflect.TypeOf((*MockServerInfoSource)(nil).ConnectedServer))
}

// Facts mocks base method
func (m *MockServerInfoSource) Facts() json.RawMessage {
	ret := m.ctrl.Call(m, "Facts")
	ret0, _ := ret[0].(json.RawMessage)
	return ret0
}

// Facts indicates an expected call of Facts
func (mr *MockServerInfoSourceMockRecorder) Facts() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Facts", reflect.TypeOf((*MockServerInfoSource)(nil).Facts))
}

// Identity mocks base method
func (m *MockServerInfoSource) Identity() string {
	ret := m.ctrl.Call(m, "Identity")
	ret0, _ := ret[0].(string)
	return ret0
}

// Identity indicates an expected call of Identity
func (mr *MockServerInfoSourceMockRecorder) Identity() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Identity", reflect.TypeOf((*MockServerInfoSource)(nil).Identity))
}

// KnownAgents mocks base method
func (m *MockServerInfoSource) KnownAgents() []string {
	ret := m.ctrl.Call(m, "KnownAgents")
	ret0, _ := ret[0].([]string)
	return ret0
}

// KnownAgents indicates an expected call of KnownAgents
func (mr *MockServerInfoSourceMockRecorder) KnownAgents() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KnownAgents", reflect.TypeOf((*MockServerInfoSource)(nil).KnownAgents))
}

// LastProcessedMessage mocks base method
func (m *MockServerInfoSource) LastProcessedMessage() time.Time {
	ret := m.ctrl.Call(m, "LastProcessedMessage")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// LastProcessedMessage indicates an expected call of LastProcessedMessage
func (mr *MockServerInfoSourceMockRecorder) LastProcessedMessage() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastProcessedMessage", reflect.TypeOf((*MockServerInfoSource)(nil).LastProcessedMessage))
}

// MachineTransition mocks base method
func (m *MockServerInfoSource) MachineTransition(name, version, path, id, transition string) error {
	ret := m.ctrl.Call(m, "MachineTransition", name, version, path, id, transition)
	ret0, _ := ret[0].(error)
	return ret0
}

// MachineTransition indicates an expected call of MachineTransition
func (mr *MockServerInfoSourceMockRecorder) MachineTransition(name, version, path, id, transition interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MachineTransition", reflect.TypeOf((*MockServerInfoSource)(nil).MachineTransition), name, version, path, id, transition)
}

// MachinesStatus mocks base method
func (m *MockServerInfoSource) MachinesStatus() ([]aagent.MachineState, error) {
	ret := m.ctrl.Call(m, "MachinesStatus")
	ret0, _ := ret[0].([]aagent.MachineState)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MachinesStatus indicates an expected call of MachinesStatus
func (mr *MockServerInfoSourceMockRecorder) MachinesStatus() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MachinesStatus", reflect.TypeOf((*MockServerInfoSource)(nil).MachinesStatus))
}

// NewEvent mocks base method
func (m *MockServerInfoSource) NewEvent(t lifecycle.Type, opts ...lifecycle.Option) error {
	varargs := []interface{}{t}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewEvent", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// NewEvent indicates an expected call of NewEvent
func (mr *MockServerInfoSourceMockRecorder) NewEvent(t interface{}, opts ...interface{}) *gomock.Call {
	varargs := append([]interface{}{t}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewEvent", reflect.TypeOf((*MockServerInfoSource)(nil).NewEvent), varargs...)
}

// Provisioning mocks base method
func (m *MockServerInfoSource) Provisioning() bool {
	ret := m.ctrl.Call(m, "Provisioning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Provisioning indicates an expected call of Provisioning
func (mr *MockServerInfoSourceMockRecorder) Provisioning() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Provisioning", reflect.TypeOf((*MockServerInfoSource)(nil).Provisioning))
}

// StartTime mocks base method
func (m *MockServerInfoSource) StartTime() time.Time {
	ret := m.ctrl.Call(m, "StartTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// StartTime indicates an expected call of StartTime
func (mr *MockServerInfoSourceMockRecorder) StartTime() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartTime", reflect.TypeOf((*MockServerInfoSource)(nil).StartTime))
}

// Stats mocks base method
func (m *MockServerInfoSource) Stats() agents.ServerStats {
	ret := m.ctrl.Call(m, "Stats")
	ret0, _ := ret[0].(agents.ServerStats)
	return ret0
}

// Stats indicates an expected call of Stats
func (mr *MockServerInfoSourceMockRecorder) Stats() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stats", reflect.TypeOf((*MockServerInfoSource)(nil).Stats))
}

// UpTime mocks base method
func (m *MockServerInfoSource) UpTime() int64 {
	ret := m.ctrl.Call(m, "UpTime")
	ret0, _ := ret[0].(int64)
	return ret0
}

// UpTime indicates an expected call of UpTime
func (mr *MockServerInfoSourceMockRecorder) UpTime() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpTime", reflect.TypeOf((*MockServerInfoSource)(nil).UpTime))
}

// PrepareForShutdown mocks base method
func (m *MockServerInfoSource) PrepareForShutdown() error {
	ret := m.ctrl.Call(m, "PrepareForShutdown")
	ret0, _ := ret[0].(error)
	return ret0
}

// PrepareForShutdown indicates an expected call of PrepareForShutdown
func (mr *MockServerInfoSourceMockRecorder) PrepareForShutdown() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrepareForShutdown", reflect.TypeOf((*MockServerInfoSource)(nil).PrepareForShutdown))
}
