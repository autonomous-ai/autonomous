package ambient

import (
	"github.com/google/wire"
)

// ProviderSet exposes ambient providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideService,
)
