package hermes

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"go.autonomous.ai/os/lib/core/system"
)

// hermesVersionProbeTimeout caps the one-shot `hermes --version` probe so a
// wedged CLI can't stall startup.
const hermesVersionProbeTimeout = 5 * time.Second

// hermesSemverRe captures the first semver-like token in `hermes --version`
// output (e.g. "Hermes Agent v0.17.0 (2026.6.19)" → "0.17.0").
var hermesSemverRe = regexp.MustCompile(`(\d+\.\d+\.\d+(?:[-+._][0-9A-Za-z.-]+)?)`)

// hermesVersion caches the parsed Hermes CLI version, populated once at startup
// by probeHermesVersion(). The agent Status endpoint (and any version surface)
// reads it via Service.Version() so the web shows the ACTIVE backend's version
// instead of OpenClaw's when Hermes is the runtime.
var hermesVersion atomic.Pointer[string]

// probeHermesVersion shells out to `hermes --version` with a short timeout and
// caches the normalized semver. Empty result (cache stays unset) when hermes is
// not on PATH or the command fails.
func probeHermesVersion() {
	ctx, cancel := context.WithTimeout(context.Background(), hermesVersionProbeTimeout)
	defer cancel()
	out, err := system.Run(ctx, "hermes", "--version")
	if err != nil {
		slog.Warn("read hermes version failed", "component", "hermes-probe", "error", err)
		return
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	v := ""
	if loc := hermesSemverRe.FindStringSubmatch(line); len(loc) > 1 {
		v = loc[1]
	}
	hermesVersion.Store(&v)
}

// Version returns the cached Hermes CLI version (e.g. "0.17.0"). Empty string
// until the startup probe completes or when hermes is not installed.
func (s *Service) Version() string {
	if v := hermesVersion.Load(); v != nil {
		return *v
	}
	return ""
}
