package healthwatch

import (
	"github.com/google/wire"
)

// ProviderSet exposes healthwatch providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideService,
)
