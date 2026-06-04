package agent

import (
	"go-lamp.autonomous.ai/domain"
	"go-lamp.autonomous.ai/internal/monitor"
	"go-lamp.autonomous.ai/internal/openclaw"
	"go-lamp.autonomous.ai/internal/statusled"
	"go-lamp.autonomous.ai/server/config"
)

// ProvideGateway returns the AgentGateway implementation based on config.AgentRuntime.
func ProvideGateway(cfg *config.Config, bus *monitor.Bus, sled *statusled.Service) domain.AgentGateway {
	switch cfg.AgentRuntime {
	default:
		return openclaw.ProvideService(cfg, bus, sled)
	}
}
