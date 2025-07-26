package cmd

import "fmt"

type stubLog struct{}

func (stubLog) DebugF(string, ...interface{}) {}
func (stubLog) InfoF(string, ...interface{})  {}
func (stubLog) WarnF(string, ...interface{})  {}
func (stubLog) ErrorF(string, ...interface{}) {}
func (stubLog) FatalF(string, ...interface{}) {}
func (stubLog) ParentID() string              { return "" }
func (stubLog) ChildID() string               { return "" }
func (stubLog) CloseLogFile()                 {}

type recordLogger struct{ infos []string }

func (recordLogger) DebugF(string, ...interface{}) {}
func (r *recordLogger) InfoF(format string, args ...interface{}) {
	r.infos = append(r.infos, fmt.Sprintf(format, args...))
}
func (recordLogger) WarnF(string, ...interface{})  {}
func (recordLogger) ErrorF(string, ...interface{}) {}
func (recordLogger) FatalF(string, ...interface{}) {}
func (recordLogger) ParentID() string              { return "" }
func (recordLogger) ChildID() string               { return "" }
func (recordLogger) CloseLogFile()                 {}
