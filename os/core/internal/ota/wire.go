package ota

import (
	"github.com/google/wire"
)

// ProviderSet exposes OTA providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideService,
)
