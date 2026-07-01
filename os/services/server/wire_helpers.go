package server

import (
	"go.autonomous.ai/os/internal/device"
	"go.autonomous.ai/os/server/config"
)

// provideStatusLEDHasLight resolves the `light` capability for the running
// device so statusled can no-op cleanly on devices without an LED. Lives in a
// non-wireinject file so both the Wire graph and the regular build compile it;
// keeping it inside wire.go (build tag wireinject) hides it from `go build`.
//
// Sits in the server pkg rather than inside statusled because statusled must
// not import the device package: device now depends on statusled for the
// wifi_connecting cue during setup, which would form an import cycle.
func provideStatusLEDHasLight(cfg *config.Config) bool {
	return device.Has(cfg.DeviceTypeOrDefault(), device.CapLight)
}
