//go:build wireinject

package server

import (
	"github.com/google/wire"

	"go.autonomous.ai/os/internal/agent"
	"go.autonomous.ai/os/internal/ambient"
	"go.autonomous.ai/os/internal/beclient"
	"go.autonomous.ai/os/internal/buddy"
	"go.autonomous.ai/os/internal/device"
	"go.autonomous.ai/os/internal/healthwatch"
	"go.autonomous.ai/os/internal/monitor"
	"go.autonomous.ai/os/internal/network"
	"go.autonomous.ai/os/internal/statusled"
	"go.autonomous.ai/os/lib/mqtt"
	"go.autonomous.ai/os/server/config"
	_buddyHttp "go.autonomous.ai/os/server/buddy/delivery/http"
	_deviceHttpDeliver "go.autonomous.ai/os/server/device/delivery/http"
	_deviceMQTTDeliver "go.autonomous.ai/os/server/device/delivery/mqtt"
	_healthHttpDeliver "go.autonomous.ai/os/server/health/delivery/http"
	_networkHttpDeliver "go.autonomous.ai/os/server/network/delivery/http"
	_agentHttp "go.autonomous.ai/os/server/agent/delivery/http"
	_sensingHttp "go.autonomous.ai/os/server/sensing/delivery/http"
)

func InitializeServer() (*Server, error) {
	panic(wire.Build(
		config.ProviderSet,
		mqtt.ProviderSet,
		beclient.ProviderSet,
		monitor.ProviderSet,
		agent.ProviderSet,
		network.ProviderSet,
		device.ProviderSet,
		buddy.ProviderSet,
		_buddyHttp.ProviderSet,
		ambient.ProviderSet,
		healthwatch.ProviderSet,
		statusled.ProviderSet,
		_healthHttpDeliver.ProviderSet,
		_networkHttpDeliver.ProviderSet,
		_deviceHttpDeliver.ProviderSet,
		_deviceMQTTDeliver.ProviderSet,
		_agentHttp.ProviderSet,
		_sensingHttp.ProviderSet,
		ProvideServer,
	))
}
