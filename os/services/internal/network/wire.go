package network

import (
	"github.com/google/wire"
)

// ProviderSet exposes network providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideService,
)
