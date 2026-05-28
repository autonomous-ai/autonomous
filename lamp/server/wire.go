//go:build wireinject

package server

import (
	"github.com/google/wire"

	"go-lamp.autonomous.ai/internal/agent"
	"go-lamp.autonomous.ai/internal/ambient"
	"go-lamp.autonomous.ai/internal/beclient"
	"go-lamp.autonomous.ai/internal/buddy"
	"go-lamp.autonomous.ai/internal/device"
	"go-lamp.autonomous.ai/internal/healthwatch"
	"go-lamp.autonomous.ai/internal/monitor"
	"go-lamp.autonomous.ai/internal/network"
	"go-lamp.autonomous.ai/internal/statusled"
	"go-lamp.autonomous.ai/lib/devicebutton"
	"go-lamp.autonomous.ai/lib/mqtt"
	"go-lamp.autonomous.ai/server/config"
	_buddyHttp "go-lamp.autonomous.ai/server/buddy/delivery/http"
	_deviceGPIODeliver "go-lamp.autonomous.ai/server/device/delivery/gpio"
	_deviceHttpDeliver "go-lamp.autonomous.ai/server/device/delivery/http"
	_deviceMQTTDeliver "go-lamp.autonomous.ai/server/device/delivery/mqtt"
	_healthHttpDeliver "go-lamp.autonomous.ai/server/health/delivery/http"
	_networkHttpDeliver "go-lamp.autonomous.ai/server/network/delivery/http"
	_agentHttp "go-lamp.autonomous.ai/server/agent/delivery/http"
	_sensingHttp "go-lamp.autonomous.ai/server/sensing/delivery/http"
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
		devicebutton.ProviderSet,
		ambient.ProviderSet,
		healthwatch.ProviderSet,
		statusled.ProviderSet,
		_healthHttpDeliver.ProviderSet,
		_networkHttpDeliver.ProviderSet,
		_deviceHttpDeliver.ProviderSet,
		_deviceMQTTDeliver.ProviderSet,
		_deviceGPIODeliver.ProviderSet,
		_agentHttp.ProviderSet,
		_sensingHttp.ProviderSet,
		ProvideServer,
	))
}
