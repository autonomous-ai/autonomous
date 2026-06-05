package device

import (
	"github.com/google/wire"
)

// ProviderSet exposes device providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideService,
)
