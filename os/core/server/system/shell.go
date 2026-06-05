package system

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// shellUpgrader allows any origin — same-origin enforcement is already handled
// at the network/proxy layer (this endpoint is only reachable on the LAN).
var shellUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// ShellHandler upgrades the request to a WebSocket and pipes a /bin/bash PTY
// in both directions. Frames from the client are stdin bytes by default; small
// JSON envelopes with `type: "resize"` resize the PTY (rows/cols).
//
// Client → server frames:
//   - TextMessage starting with '{' and ending with '}' AND parseable as
//     {"type":"resize","rows":N,"cols":M}  ⇒ window resize signal
//   - Anything else (text or binary)       ⇒ raw stdin bytes
//
// Server → client frames: raw stdout/stderr bytes as binary messages.
func ShellHandler(c *gin.Context) {
	ws, err := shellUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[shell] upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	// Spawn an interactive login bash so the user gets aliases, $PATH, prompt.
	cmd := exec.Command("/bin/bash", "-il")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[shell] pty start failed: %v", err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n[shell] failed to start PTY: "+err.Error()+"\r\n"))
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	}()

	// Initial size — client will send a resize frame on connect to override.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	// One writer mutex: WebSocket connections require all writes to be serialized.
	var writeMu sync.Mutex
	writeBytes := func(t int, b []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return ws.WriteMessage(t, b)
	}

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	// PTY → WebSocket. Read in 4KB chunks; xterm.js handles ANSI just fine.
	go func() {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := writeBytes(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("[shell] pty read: %v", err)
				}
				return
			}
		}
	}()

	// WebSocket → PTY. Loop ends when the client closes or we get an error.
	for {
		select {
		case <-done:
			return
		default:
		}
		mt, data, err := ws.ReadMessage()
		if err != nil {
			closeDone()
			return
		}

		// Try to interpret as a control envelope (resize) — only for text frames
		// that look like JSON. Anything else goes straight to PTY stdin.
		if mt == websocket.TextMessage && len(data) > 1 && data[0] == '{' {
			var env struct {
				Type string `json:"type"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if jerr := json.Unmarshal(data, &env); jerr == nil && env.Type == "resize" {
				if env.Rows == 0 {
					env.Rows = 24
				}
				if env.Cols == 0 {
					env.Cols = 80
				}
				_ = pty.Setsize(ptmx, &pty.Winsize{Rows: env.Rows, Cols: env.Cols})
				continue
			}
		}

		if _, werr := ptmx.Write(data); werr != nil {
			log.Printf("[shell] pty write: %v", werr)
			closeDone()
			return
		}
	}
}
