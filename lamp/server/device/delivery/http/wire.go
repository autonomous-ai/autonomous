package http

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	ProvideDeviceHandler,
)
