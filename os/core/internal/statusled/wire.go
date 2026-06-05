package statusled

import "github.com/google/wire"

// ProviderSet exposes statusled providers for Wire.
var ProviderSet = wire.NewSet(ProvideService)
