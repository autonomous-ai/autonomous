package safego

import (
	"log/slog"
	"runtime/debug"
)

// Go launches fn in a goroutine with panic recovery.
// If fn panics, the panic is logged and the goroutine exits cleanly
// instead of crashing the entire process.
func Go(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panic recovered",
					"component", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}
