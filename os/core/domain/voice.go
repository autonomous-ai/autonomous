package domain

import (
	"strings"
	"sync"
	"time"
)

const (
	// enrollNudgeCooldown prevents spamming "who are you?" for the same unknown speaker.
	enrollNudgeCooldown = 5 * time.Minute
	enrollInstruction   = "\n[REQUIRED: Follow speaker-recognizer/SKILL.md — check if user is introducing themselves. If yes, enroll voice immediately.]"
)

var (
	lastEnrollNudge   time.Time
	lastEnrollNudgeMu sync.Mutex
)

// AppendEnrollNudge checks if a voice message is from an unknown speaker with
// saved audio, and appends the enroll instruction if cooldown has elapsed.
// Returns the message unchanged if not applicable.
func AppendEnrollNudge(msg string) string {
	if !strings.Contains(msg, "Unknown Speaker:") || !strings.Contains(msg, "audio save at") {
		return msg
	}

	lastEnrollNudgeMu.Lock()
	defer lastEnrollNudgeMu.Unlock()

	if time.Since(lastEnrollNudge) < enrollNudgeCooldown {
		return msg
	}
	lastEnrollNudge = time.Now()
	return msg + enrollInstruction
}
