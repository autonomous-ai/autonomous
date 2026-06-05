package agent

import (
	"go.autonomous.ai/os/domain"
	"go.autonomous.ai/os/internal/monitor"
	"go.autonomous.ai/os/internal/openclaw"
	"go.autonomous.ai/os/internal/statusled"
	"go.autonomous.ai/os/server/config"
)

// ProvideGateway returns the AgentGateway implementation based on config.AgentRuntime.
func ProvideGateway(cfg *config.Config, bus *monitor.Bus, sled *statusled.Service) domain.AgentGateway {
	switch cfg.AgentRuntime {
	default:
		return openclaw.ProvideService(cfg, bus, sled)
	}
}
