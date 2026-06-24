package picoclaw

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// gatewayRestartTimeout bounds a single `systemctl restart picoclaw`.
const gatewayRestartTimeout = 60 * time.Second

// restartPicoclawGateway restarts the picoclaw systemd unit so the gateway re-reads
// changed workspace prompt files. Mirrors openclaw's restartOpenclawGateway (which
// lives in internal/openclaw/service_gateway.go), but the picoclaw gateway has no
// foreground-friendly `restart` subcommand, so when systemctl is unavailable
// (non-root / dev box) we log and skip rather than hard-fail.
func restartPicoclawGateway() error {
	ctx, cancel := context.WithTimeout(context.Background(), gatewayRestartTimeout)
	defer cancel()

	if os.Geteuid() == 0 {
		if _, err := exec.LookPath("systemctl"); err == nil {
			out, err := exec.CommandContext(ctx, "systemctl", "restart", "picoclaw").CombinedOutput()
			if err == nil {
				return nil
			}
			slog.Warn("systemctl restart picoclaw failed", "component", "picoclaw-onboarding",
				"output", strings.TrimSpace(string(out)))
		}
	}
	// TODO(picoclaw-reload): when systemctl is unavailable, fall back to POST
	// http://localhost:18790/reload (the gateway exposes it) instead of skipping.
	slog.Warn("no systemctl restart available — skipping picoclaw gateway restart (changes apply on next start)",
		"component", "picoclaw-onboarding")
	return nil
}
