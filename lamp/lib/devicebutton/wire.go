package devicebutton

import (
	"github.com/google/wire"
)

// ProviderSet exposes reset button providers for Wire.
// ProvideServiceOptional returns *Service (nil when GPIO unavailable, e.g. dev machine).
var ProviderSet = wire.NewSet(
	ProvideDeviceButtonOptional,
)

// ProvideDeviceButtonOptional returns a DeviceButton with GPIO initialized;
// returns nil when GPIO is unavailable (e.g. dev machine) so server can still start.
func ProvideDeviceButtonOptional() *DeviceButton {
	btn := ProvideDeviceButton()
	if err := btn.Init(); err != nil {
		return nil
	}
	return btn
}
