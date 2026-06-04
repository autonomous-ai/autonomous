package beclient

import (
	"github.com/google/wire"
	"go-lamp.autonomous.ai/server/config"
)

// ProviderSet exposes beclient providers for Wire.
var ProviderSet = wire.NewSet(
	ProvideClient,
)

// ProvideClient creates a BE client. Base URL is read from config.LLMBaseURL on each Ping.
func ProvideClient(cfg *config.Config) *Client {
	return New(cfg)
}
