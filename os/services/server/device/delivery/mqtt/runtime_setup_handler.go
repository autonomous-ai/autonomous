package mqtthandler

import (
	"log/slog"

	"go.autonomous.ai/os/domain"
	"go.autonomous.ai/os/internal/device"
)

// handleRuntimeSetup applies a `hermes.setup` / `picoclaw.setup` downlink — swap
// the active agentic backend. The kind itself names the target runtime (passed
// in by the dispatcher), so unlike the former generic agent_runtime.set there is
// no runtime field to read off the wire.
//
// Flow: ack "starting" immediately, then in a goroutine run the switch and WAIT
// for its real outcome. UpdateAgentRuntime blocks until switch-runtime finishes,
// so the success/failure ack here reflects what actually happened — not an
// optimistic guess. On a confirmed switch we ack "success" and THEN restart
// os-server (the ack must reach the wire first, since the restart kills us); the
// worker sees the swap land via the brief reconnect + new AGENT BACKEND ACTIVE
// banner. On failure switch-runtime has already rolled back, so we ack "failure".
// Every ack echoes the triggering kind so the worker can match hermes.setup vs
// picoclaw.setup.

func (h *DeviceMQTTHandler) publishRuntimeSetupAck(kind, status, errMsg string, data *domain.AgentRuntimeSetData) {
	ack := domain.AgentRuntimeSetAck{
		MQTTInfoResponse: domain.NewMQTTInfoResponse(h.config, "data", device.GetDeviceMac()),
		Kind:             kind,
		Status:           status,
		Error:            errMsg,
		Data:             data,
	}
	if err := h.publish(ack); err != nil {
		slog.Warn("runtime setup: publish ack failed", "component", "mqtt", "kind", kind, "status", status, "error", err)
	}
}

// handleRuntimeSetup is shared by the hermes.setup and picoclaw.setup dispatch
// cases; runtime is the target backend named by the kind.
func (h *DeviceMQTTHandler) handleRuntimeSetup(env domain.MQTTDataCommand, runtime string) error {
	kind := env.Kind
	req := domain.AgentRuntimeSetData{Runtime: runtime}

	slog.Info("runtime setup: received", "component", "mqtt", "kind", kind, "runtime", runtime)

	// Ack immediately so the worker knows the device received the command.
	h.publishRuntimeSetupAck(kind, "starting", "", nil)

	go func() {
		switched, err := h.deviceService.UpdateAgentRuntime(req)
		if err != nil {
			// switch-runtime already rolled back; report the real failure.
			slog.Error("runtime setup: switch failed", "component", "mqtt", "kind", kind, "error", err)
			h.publishRuntimeSetupAck(kind, "failure", err.Error(), &req)
			return
		}
		// Switch confirmed landed (or was a no-op). Ack success — it must reach the
		// wire BEFORE the os-server restart below, which kills us.
		slog.Info("runtime setup: switch confirmed", "component", "mqtt", "kind", kind, "runtime", runtime, "switched", switched)
		h.publishRuntimeSetupAck(kind, "success", "", &req)

		if switched {
			// Restart os-server so factory.go re-resolves the gateway to the new
			// backend. Deferred until after the ack on purpose.
			if rerr := h.deviceService.RestartForAgentRuntime(); rerr != nil {
				slog.Error("runtime setup: os-server restart failed", "component", "mqtt", "kind", kind, "error", rerr)
			}
		}
	}()

	return nil
}
