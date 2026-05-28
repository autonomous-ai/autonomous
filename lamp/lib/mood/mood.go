// Package mood provides a per-user mood history logger.
//
// Tracks the user's emotional state over time. Each row is one of two kinds:
//   - "signal"   — raw evidence from a single source (camera/voice/telegram).
//   - "decision" — agent-synthesized mood derived from recent signals + last
//     decision. Carries BasedOn and Reasoning for traceability.
//
// The agent owns mood synthesis. The store only persists rows; consumers query
// `kind=decision&last=1` for the current mood, or `kind=signal` to re-analyze.
//
// Usage:
//
//	mood.Init()                                                   // once at startup
//	mood.SetCurrentUser("gray")                                   // on presence.enter
//	mood.LogSignal("gray", "happy", "camera", "laughing")         // raw signal
//	mood.LogDecision("gray", "stressed", "5 signals last 30min", "yawning + work complaints")
//	events := mood.Query("gray", "2026-04-07", "", 100)           // all kinds
//	last := mood.Query("gray", "2026-04-07", "decision", 1)       // latest decision
//	mood.ClearCurrentUser()                                       // on presence.leave
package mood

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-lamp.autonomous.ai/lib/usercanon"
)

// Event is one mood history record persisted to JSONL.
type Event struct {
	TS        float64 `json:"ts"`                  // Unix seconds
	Seq       int64   `json:"seq"`                 // global sequence
	Hour      int     `json:"hour"`                // hour of day (0-23)
	Kind      string  `json:"kind"`                // "signal" (raw) or "decision" (agent-synthesized)
	Mood      string  `json:"mood"`                // user mood: happy, sad, stressed, tired, excited, etc.
	Source    string  `json:"source,omitempty"`    // signal: camera|voice|telegram|conversation. decision: "agent"
	Trigger   string  `json:"trigger,omitempty"`   // signal: action/context. decision: omit.
	BasedOn   string  `json:"based_on,omitempty"`  // decision only: short summary of inputs (e.g. "5 signals last 30min")
	Reasoning string  `json:"reasoning,omitempty"` // decision only: why this mood was chosen
}

// Event kinds.
const (
	KindSignal   = "signal"
	KindDecision = "decision"
)

const (
	usersDir      = "/root/local/users"
	moodSubdir    = "mood"
	fileSuffix    = ".jsonl"
	retentionDays = 30
	DefaultUser   = "unknown"
)

type logger struct {
	mu   sync.Mutex
	seqN atomic.Int64
	file *os.File
	day  string
	user string // current file's user

	// Current user tracking
	currentUserMu sync.RWMutex
	currentUser   string
}

var global = &logger{}

// Init creates the users directory.
// Call once at startup.
func Init() {
	_ = os.MkdirAll(usersDir, 0o755)
	go cleanOldLogs()
}

// SetCurrentUser sets the user who is currently present.
// Call on presence.enter with the recognized user name.
func SetCurrentUser(name string) {
	resolved := usercanon.Resolve(name)
	global.currentUserMu.Lock()
	global.currentUser = resolved
	global.currentUserMu.Unlock()
	slog.Info("mood: current user set", "user", resolved, "raw", name)
}

// ClearCurrentUser clears the current user.
// Call on presence.leave.
func ClearCurrentUser() {
	global.currentUserMu.Lock()
	global.currentUser = ""
	global.currentUserMu.Unlock()
}

// CurrentUser returns the current user name (empty if none).
func CurrentUser() string {
	global.currentUserMu.RLock()
	defer global.currentUserMu.RUnlock()
	return global.currentUser
}

// LogMood records a raw mood signal for the current user (presence-detected).
// Falls back to "unknown" when no user is detected via presence.
func LogMood(moodStr, source, trigger string) {
	user := CurrentUser()
	if user == "" {
		user = DefaultUser
	}
	LogMoodForUser(user, moodStr, source, trigger)
}

// LogMoodForUser records a raw mood signal for a specific user.
// Equivalent to LogSignal — kept for backward compatibility.
func LogMoodForUser(user, moodStr, source, trigger string) {
	LogSignal(user, moodStr, source, trigger)
}

// LogSignal appends a raw signal row.
func LogSignal(user, moodStr, source, trigger string) {
	writeEvent(user, Event{
		Kind:    KindSignal,
		Mood:    moodStr,
		Source:  source,
		Trigger: trigger,
	})
}

// LogDecision appends an agent-synthesized decision row.
// basedOn is a short summary of the inputs the agent considered.
// reasoning explains why this mood was chosen.
func LogDecision(user, moodStr, basedOn, reasoning string) {
	writeEvent(user, Event{
		Kind:      KindDecision,
		Mood:      moodStr,
		Source:    "agent",
		BasedOn:   basedOn,
		Reasoning: reasoning,
	})
}

// LogEvent appends a fully-formed event. Use this from HTTP handlers that
// already know all fields. TS/Seq/Hour are filled if zero. Kind defaults to
// "signal" when blank.
func LogEvent(user string, evt Event) {
	if evt.Kind == "" {
		evt.Kind = KindSignal
	}
	writeEvent(user, evt)
}

func writeEvent(user string, evt Event) {
	now := time.Now()
	evt.TS = float64(now.UnixNano()) / 1e9
	evt.Seq = global.seqN.Add(1)
	evt.Hour = now.Hour()

	global.mu.Lock()
	global.writeJSONL(now, user, evt)
	global.mu.Unlock()
}

// Query reads mood events for a given user and day (YYYY-MM-DD format).
// kind filters by Event.Kind ("signal" or "decision"). Empty string returns all.
// Returns up to last n events (after filtering). If n <= 0, returns all.
func Query(user string, day string, kind string, n int) []Event {
	path := moodFilePath(user, day)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return nil
	}

	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		// Backfill Kind for legacy rows written before the field existed.
		if evt.Kind == "" {
			evt.Kind = KindSignal
		}
		if kind != "" && evt.Kind != kind {
			continue
		}
		events = append(events, evt)
	}

	if n > 0 && len(events) > n {
		events = events[len(events)-n:]
	}
	return events
}

// moodFilePath returns the JSONL file path for a user+day.
func moodFilePath(user, day string) string {
	return filepath.Join(usersDir, user, moodSubdir, day+fileSuffix)
}

// writeJSONL appends the event to the user's daily JSONL file.
// Must be called with mu held.
func (l *logger) writeJSONL(now time.Time, user string, evt Event) {
	day := now.Format("2006-01-02")

	// Reopen file if day or user changed
	if l.day != day || l.user != user || l.file == nil {
		if l.file != nil {
			_ = l.file.Close()
		}
		path := moodFilePath(user, day)
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			slog.Error("mood: failed to open log file", "path", path, "error", err)
			l.file = nil
			return
		}
		l.file = f
		l.day = day
		l.user = user
	}
	b, _ := json.Marshal(evt)
	_, _ = l.file.Write(append(b, '\n'))
}

func cleanOldLogs() {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Format("2006-01-02")

	userDirs, err := os.ReadDir(usersDir)
	if err != nil {
		return
	}
	for _, ud := range userDirs {
		if !ud.IsDir() || strings.HasPrefix(ud.Name(), ".") {
			continue
		}
		moodDir := filepath.Join(usersDir, ud.Name(), moodSubdir)
		cleanDir(moodDir, cutoff)
	}
}

func cleanDir(dir, cutoff string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, fileSuffix) {
			continue
		}
		date := strings.TrimSuffix(name, fileSuffix)
		if date < cutoff {
			path := filepath.Join(dir, name)
			if err := os.Remove(path); err == nil {
				slog.Info("removed old mood log", "component", "mood", "file", path)
			}
		}
	}
}
