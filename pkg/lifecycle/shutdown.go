package lifecycle

import "sync"

var (
	shutdownFn func() error
	closeFn    func()
	mu         sync.Mutex
)

// RegisterShutdown sets the function to execute when a shutdown is triggered.
func RegisterShutdown(fn func() error) {
	mu.Lock()
	defer mu.Unlock()
	shutdownFn = fn
}

// RegisterClose sets the function to execute to immediately stop accepting new
// connections. It is safe to call this multiple times, the last registration wins.
func RegisterClose(fn func()) {
	mu.Lock()
	defer mu.Unlock()
	closeFn = fn
}

// CloseServerListener executes the registered close function if present.
func CloseServerListener() {
	mu.Lock()
	fn := closeFn
	mu.Unlock()
	if fn != nil {
		fn()
	}
}

// ShutdownServer runs the registered shutdown function if set.
func ShutdownServer() error {
	mu.Lock()
	fn := shutdownFn
	mu.Unlock()
	if fn != nil {
		return fn()
	}
	return nil
}
