package mqtthandler

import (
	"log/slog"

	"go.autonomous.ai/os/domain"
	"go.autonomous.ai/os/internal/device"
	"go.autonomous.ai/os/lib/hal"
	agenthttp "go.autonomous.ai/os/server/agent/delivery/http"
)

func (h *DeviceMQTTHandler) handleInfo(_ domain.MQTTMessage) error {
	msg := domain.NewMQTTInfoResponse(h.config, "info", device.GetDeviceMac())
	if v, err := hal.GetVersion(); err == nil {
		msg.HalVersion = v
	}
	msg.OpenClawVersion = agenthttp.GetOpenClawVersion()
	msg.AgentRuntime = device.CurrentAgentRuntimeFromConfig(h.config)
	// Version of whatever backend is actually running (openclaw → its cache,
	// hermes → `hermes --version`), so the info message reports the live agent's
	// version the same way it always has for openclaw.
	msg.AgentVersion = h.agentGateway.Version()
	if ip, err := h.networkService.GetCurrentIP(); err == nil {
		msg.LocalIP = ip
	}
	slog.Info("mqtt_handler_info",
		"id", msg.ID,
		"version", msg.Version,
		"hal_version", msg.HalVersion,
		"openclaw_version", msg.OpenClawVersion,
		"agent_runtime", msg.AgentRuntime,
		"agent_version", msg.AgentVersion,
		"local_ip", msg.LocalIP,
		"tts_provider", msg.TTSProvider,
		"tts_voice", msg.TTSVoice,
		"stt_language", msg.STTLanguage,
	)
	return h.publish(msg)
}
