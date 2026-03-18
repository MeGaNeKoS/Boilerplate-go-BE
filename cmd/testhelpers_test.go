package cmd

import "fmt"

type stubLog struct{}

func (stubLog) Debug(string)                  {}
func (stubLog) DebugF(string, ...interface{}) {}
func (stubLog) Info(string)                    {}
func (stubLog) InfoF(string, ...interface{})  {}
func (stubLog) Warn(string)                    {}
func (stubLog) WarnF(string, ...interface{})  {}
func (stubLog) Error(string)                   {}
func (stubLog) ErrorF(string, ...interface{}) {}
func (stubLog) Fatal(string)                   {}
func (stubLog) FatalF(string, ...interface{}) {}
func (stubLog) ParentID() string              { return "" }
func (stubLog) ChildID() string               { return "" }
func (stubLog) CloseLogFile()                 {}

type recordLogger struct{ infos []string }

func (recordLogger) Debug(string)                  {}
func (recordLogger) DebugF(string, ...interface{}) {}
func (recordLogger) Info(string)                    {}
func (r *recordLogger) InfoF(format string, args ...interface{}) {
	r.infos = append(r.infos, fmt.Sprintf(format, args...))
}
func (recordLogger) Warn(string)                    {}
func (recordLogger) WarnF(string, ...interface{})  {}
func (recordLogger) Error(string)                   {}
func (recordLogger) ErrorF(string, ...interface{}) {}
func (recordLogger) Fatal(string)                   {}
func (recordLogger) FatalF(string, ...interface{}) {}
func (recordLogger) ParentID() string              { return "" }
func (recordLogger) ChildID() string               { return "" }
func (recordLogger) CloseLogFile()                 {}
