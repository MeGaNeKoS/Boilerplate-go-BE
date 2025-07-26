package example

// stubLogger is a minimal logger implementation returning fixed parent ID.
type stubLogger struct{ parent string }

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (s stubLogger) ParentID() string            { return s.parent }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}
