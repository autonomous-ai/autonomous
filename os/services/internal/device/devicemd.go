package device

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reFrontMatter     = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	reGatewayBlock    = regexp.MustCompile(`(?m)^gateway:[ \t]*\n((?:[ \t]+.*\n?)+)`)
	reGatewayDefault  = regexp.MustCompile(`(?m)^[ \t]+default:[ \t]*(\S+)`)
	reGatewayProtocol = regexp.MustCompile(`(?m)^[ \t]+protocol:[ \t]*(\S+)`)
	reSoulRef         = regexp.MustCompile(`(?m)^soul_ref:[ \t]*(\S+)`)
	reCapBlock        = regexp.MustCompile(`(?m)^capabilities:[ \t]*\n((?:[ \t]+.*\n?)+)`)
	reCapKey          = regexp.MustCompile(`(?m)^[ \t]+(\w+):`)
)

// Capabilities returns the set of capability keys declared in the
// `capabilities:` block of devices/<deviceType>/DEVICE.md (e.g. audio, vision,
// motion, light, display, …), or nil if absent/unreadable. Dependency-free
// front-matter parse, mirroring SoulRef/GatewayDefault. The capability keys are
// what gate which hardware/body skills a device loads (see openclaw onboarding):
// a skill that declares `capability: motion` is only shipped to a device whose
// DEVICE.md declares `motion`.
func Capabilities(deviceType string) map[string]bool {
	b, err := os.ReadFile(filepath.Join(DevicesDir(), deviceType, "DEVICE.md"))
	if err != nil {
		return nil
	}
	fm := reFrontMatter.FindSubmatch(b)
	if fm == nil {
		return nil
	}
	blk := reCapBlock.FindSubmatch(fm[1])
	if blk == nil {
		return nil
	}
	caps := map[string]bool{}
	for _, m := range reCapKey.FindAllSubmatch(blk[1], -1) {
		caps[strings.TrimSpace(string(m[1]))] = true
	}
	return caps
}

// SoulRef returns the `soul_ref` declared in devices/<deviceType>/DEVICE.md, or
// "" if absent/unreadable. The value is either a path (read relative to the
// device dir) or an http(s) URL (downloaded) — see openclaw.deviceSoulCore.
// Dependency-free front-matter parse, mirroring GatewayDefault.
func SoulRef(deviceType string) string {
	b, err := os.ReadFile(filepath.Join(DevicesDir(), deviceType, "DEVICE.md"))
	if err != nil {
		return ""
	}
	fm := reFrontMatter.FindSubmatch(b)
	if fm == nil {
		return ""
	}
	m := reSoulRef.FindSubmatch(fm[1])
	if m == nil {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}

// DevicesDir resolves the per-device profile root (devices/<type>/...).
// DEVICES_DIR env wins; falls back to /opt/devices (mirrors HAL + onboarding.go).
func DevicesDir() string {
	if d := os.Getenv("DEVICES_DIR"); d != "" {
		return d
	}
	return "/opt/devices"
}

// gatewayField extracts one sub-field (matched by re) from the `gateway:` block
// of devices/<deviceType>/DEVICE.md, or "" if absent/unreadable. Dependency-free
// front-matter parse (no YAML lib), mirroring hal/board/device.py.
func gatewayField(deviceType string, re *regexp.Regexp) string {
	b, err := os.ReadFile(filepath.Join(DevicesDir(), deviceType, "DEVICE.md"))
	if err != nil {
		return ""
	}
	fm := reFrontMatter.FindSubmatch(b)
	if fm == nil {
		return ""
	}
	blk := reGatewayBlock.FindSubmatch(fm[1])
	if blk == nil {
		return ""
	}
	m := re.FindSubmatch(blk[1])
	if m == nil {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}

// GatewayDefault returns the `gateway.default` (agentic runtime) declared in
// devices/<deviceType>/DEVICE.md, or "" if absent.
func GatewayDefault(deviceType string) string {
	return gatewayField(deviceType, reGatewayDefault)
}

// GatewayProtocol returns the `gateway.protocol` (wire transport) declared in
// devices/<deviceType>/DEVICE.md, or "" if absent. The transport is actually a
// property of the runtime (openclaw→websocket, hermes→sse), so this is consumed
// only as a consistency guard — see agent.ProvideGateway.
func GatewayProtocol(deviceType string) string {
	return gatewayField(deviceType, reGatewayProtocol)
}
