package agent

import (
	"log/slog"

	"go.autonomous.ai/os/domain"
	"go.autonomous.ai/os/internal/device"
	"go.autonomous.ai/os/internal/hermes"
	"go.autonomous.ai/os/internal/monitor"
	"go.autonomous.ai/os/internal/openclaw"
	"go.autonomous.ai/os/internal/statusled"
	"go.autonomous.ai/os/server/config"
)

// ProvideGateway returns the AgentGateway implementation. The backend is chosen
// by config.AgentRuntime; when that is unset it falls back to the device's
// declared gateway.default (devices/<type>/DEVICE.md), then OpenClaw.
//
// "openclaw" (default): persistent WebSocket to the OpenClaw daemon at
// 127.0.0.1:18789. See internal/openclaw and docs/os-server.md.
//
// "hermes": HTTP+SSE client against the Hermes API server (default
// 127.0.0.1:8642). See internal/hermes and docs/hermes.md.
func ProvideGateway(cfg *config.Config, bus *monitor.Bus, sled *statusled.Service) domain.AgentGateway {
	// Backend resolution: explicit config.agent_runtime wins; if unset, fall back
	// to the device's declared gateway.default (DEVICE.md); else OpenClaw default.
	runtime := cfg.AgentRuntime
	source := "config.agent_runtime"
	if runtime == "" {
		if g := device.GatewayDefault(cfg.DeviceTypeOrDefault()); g != "" {
			runtime, source = g, "DEVICE.md gateway.default"
		}
	}

	switch runtime {
	case "hermes":
		logBackendBanner("HERMES", map[string]string{
			"base_url":     hermes.BaseURL,
			"conversation": hermes.Conversation,
			"model":        hermes.Model,
			"api_key_set":  boolStr(hermes.APIKey != ""),
			"source":       source,
		})
		return hermes.ProvideService(cfg, bus, sled)
	default:
		effective := runtime
		if effective == "" {
			effective = "openclaw (default — agent_runtime + gateway.default both unset)"
		} else if effective != "openclaw" {
			effective = "openclaw (FALLBACK — unknown runtime=" + runtime + ")"
		}
		logBackendBanner("OPENCLAW", map[string]string{
			"config_dir":      cfg.OpenclawConfigDir,
			"effective_value": effective,
			"source":          source,
		})
		return openclaw.ProvideService(cfg, bus, sled)
	}
}

func logBackendBanner(name string, fields map[string]string) {
	args := []any{"component", "agent", "backend", name}
	for k, v := range fields {
		args = append(args, k, v)
	}
	slog.Info("══════════════════════════════════════════════════════", "component", "agent")
	slog.Info("  AGENT BACKEND ACTIVE → "+name, args...)
	slog.Info("══════════════════════════════════════════════════════", "component", "agent")
}

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
