//go:build !systemd

package supervisor

// RequestRestart is a no-op when the systemd build tag is not enabled.
//
//go:noinline
func RequestRestart() error { return nil }

// OnStartUp is a no-op when not running under systemd.
//
//go:noinline
func OnStartUp() error { return nil }
