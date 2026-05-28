package http

import (
	"github.com/google/wire"
)

// ProviderSet exposes the buddy HTTP handler for Wire.
var ProviderSet = wire.NewSet(
	ProvideBuddyHandler,
)
