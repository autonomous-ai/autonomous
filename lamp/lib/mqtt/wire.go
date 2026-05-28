package mqtt

import "github.com/google/wire"

// ProviderSet exposes the MQTT client provider for Wire.
// ProvideClient builds *MQTT from Config (caller must provide Config, e.g. from server config).
var ProviderSet = wire.NewSet(
	ProvideFactory,
	ProvideClient,
)
