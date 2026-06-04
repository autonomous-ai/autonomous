package http

import (
	"log/slog"
	"os/exec"
	"time"

	"go-lamp.autonomous.ai/domain"
	"go-lamp.autonomous.ai/internal/network"
	"go-lamp.autonomous.ai/server/config"
)

const (
	powerOffThreshold     = 3 * time.Second
	factoryResetThreshold = 10 * time.Second
)

// DeviceGPIOHandler represents the GPIO handler for device
type DeviceGPIOHandler struct {
	config         *config.Config
	networkService *network.Service
	agentGateway   domain.AgentGateway
}

func ProvideDeviceGPIOHandler(config *config.Config, networkService *network.Service, gw domain.AgentGateway) DeviceGPIOHandler {
	return DeviceGPIOHandler{
		config:         config,
		networkService: networkService,
		agentGateway:   gw,
	}
}

// HandlePress is called on short tap of the device button.
func (h *DeviceGPIOHandler) HandlePress() {
	slog.Info("restarting agent", "component", "device-button")
	if err := h.agentGateway.RestartAgent(); err != nil {
		slog.Error("restart agent failed", "component", "device-button", "error", err)
		return
	}
	slog.Info("restart agent done", "component", "device-button")
}

// HandlePressAndHold is called while the button is held (released=false) and once
// on release (released=true). It interprets duration to trigger power-off or factory-reset.
func (h *DeviceGPIOHandler) HandlePressAndHold(duration time.Duration, released bool) {
	if !released {
		// Threshold feedback while still holding
		switch {
		case duration >= factoryResetThreshold && duration < factoryResetThreshold+time.Second:
			slog.Info("factory reset threshold reached", "component", "device-button", "duration", duration)
		case duration >= powerOffThreshold && duration < powerOffThreshold+time.Second:
			slog.Info("power off threshold reached", "component", "device-button", "duration", duration)
		}
		return
	}

	// Released — fire the appropriate action
	if duration >= factoryResetThreshold {
		h.factoryReset()
	} else if duration >= powerOffThreshold {
		h.powerOff()
	}
}

func (h *DeviceGPIOHandler) powerOff() {
	slog.Info("power off", "component", "device-button")
	if err := exec.Command("systemctl", "poweroff").Run(); err != nil {
		slog.Error("power off failed", "component", "device-button", "error", err)
	}
}

func (h *DeviceGPIOHandler) factoryReset() {
	slog.Info("factory reset", "component", "device-button")
	if err := h.agentGateway.ResetAgent(); err != nil {
		slog.Error("reset agent failed", "component", "device-button", "error", err)
		return
	}
	if err := h.config.ResetToDefault(); err != nil {
		slog.Error("factory reset failed", "component", "device-button", "error", err)
		return
	}
	if err := h.networkService.ResetNetwork(); err != nil {
		slog.Error("reset network failed", "component", "device-button", "error", err)
		return
	}
	if err := h.networkService.SwitchToAPMode(); err != nil {
		slog.Error("switch to AP mode failed", "component", "device-button", "error", err)
	}
	slog.Info("config reset to default (factory reset)", "component", "device-button")
}
