package domain

// OTAComponent describes version and download URL for a single component.
type OTAComponent struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

const (
	OTAKeyOSServer  = "os-server"
	OTAKeyBootstrap = "bootstrap"
	OTAKeyOpenClaw  = "openclaw"
	OTAKeyWeb       = "web"
	// OTAKeyHal's value is "hal" — the OTA metadata key, on-device deploy
	// dir (/opt/hal), and `software-update` arg.
	OTAKeyHal   = "hal"
	OTAKeyBuddy = "claude-desktop-buddy"
	// OTAKeyDevice is the on-device `software-update` arg for the device profile.
	// Unlike the others it is NOT a flat metadata key — the profile lives nested
	// at metadata.devices.<device_type> (one metadata.json serves all types).
	OTAKeyDevice = "device"
)

// OTAMetadata is the JSON shape returned by the OTA metadata URL.
//
// Example:
//
//	{
//	  "lamp":    {"version":"1.2.3","url":"https://..."},
//	  "bootstrap": {"version":"2.3.4","url":"https://..."},
//	  "web":      {"version":"0.9.0","url":"https://..."}
//	}
type OTAMetadata map[string]OTAComponent
