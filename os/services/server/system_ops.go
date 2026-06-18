package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/server/serializers"
)

// softwareUpdateLastFire tracks the last time each OTA target was triggered, so
// a stuck/looping caller can't kick off back-to-back force-checks. Bootstrap's
// downloader is idempotent but the resulting service restarts (os-server +
// systemd reload + journal noise) are not free; 30 s is enough to absorb a
// double-click without hiding genuine retries.
var (
	softwareUpdateLastFire   = map[string]time.Time{}
	softwareUpdateLastFireMu sync.Mutex
)

const softwareUpdateMinInterval = 30 * time.Second

// softwareUpdate triggers an OTA update for a single named component via the bootstrap worker.
// POST /api/system/software-update/:target  (target: os-server | web | hal)
func (s *Server) softwareUpdate(c *gin.Context) {
	target := c.Param("target")
	allowed := map[string]bool{"os-server": true, "web": true, "hal": true}
	if !allowed[target] {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("unknown target: "+target))
		return
	}

	// Per-target rate limit. Returns 429 with retry-after so the web button
	// can surface a useful message instead of looking broken.
	softwareUpdateLastFireMu.Lock()
	if last, ok := softwareUpdateLastFire[target]; ok {
		if wait := softwareUpdateMinInterval - time.Since(last); wait > 0 {
			softwareUpdateLastFireMu.Unlock()
			c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			c.JSON(http.StatusTooManyRequests,
				serializers.ResponseError(fmt.Sprintf("software-update %s rate-limited, retry in %ds", target, int(wait.Seconds())+1)))
			return
		}
	}
	softwareUpdateLastFire[target] = time.Now()
	softwareUpdateLastFireMu.Unlock()

	url := "http://127.0.0.1:8080/force-check/" + target
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializers.ResponseError("build request: "+err.Error()))
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, serializers.ResponseError("bootstrap unreachable: "+err.Error()))
		return
	}
	defer resp.Body.Close()
	c.JSON(http.StatusOK, serializers.ResponseSuccess("software update triggered: "+target))
}

// execCommand runs a shell command (sh -c) and returns stdout, stderr, and exit code.
// POST /api/system/exec  body: {"cmd": "..."}
func (s *Server) execCommand(c *gin.Context) {
	var body struct {
		Cmd string `json:"cmd"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Cmd) == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("cmd required"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", body.Cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			if stderr.Len() == 0 {
				stderr.WriteString(err.Error())
			}
		}
	}

	c.JSON(http.StatusOK, serializers.ResponseSuccess(map[string]any{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
	}))
}
