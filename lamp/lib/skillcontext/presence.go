package skillcontext

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go-lamp.autonomous.ai/lib/usercanon"
	"go-lamp.autonomous.ai/lib/wellbeing"
)

// presenceLookbackDays caps how far back LastActionTS scans for the most
// recent `leave` row. 3 days covers a long weekend away while keeping file
// reads bounded (retention is 7 days anyway, so going further would only
// matter on the edge of cleanup).
const presenceLookbackDays = 3

// presenceContext is the digest the agent reads on presence.enter events.
// Pre-computes "how long since the user was last seen" so sensing/SKILL.md
// can swap to a return-after-long-absence phrasing without re-fetching the
// wellbeing log itself.
type presenceContext struct {
	// LastLeaveAgeMin is minutes since the most recent `leave` row in the
	// user's wellbeing log across the last presenceLookbackDays days. -1
	// when no leave row exists in that window (first session ever, or
	// retention cleared older data).
	LastLeaveAgeMin int `json:"last_leave_age_min"`
	// CurrentHour is the current hour 0-23 — lets the SKILL gate the
	// return-welcome route on time-of-day without parsing the timestamp.
	CurrentHour int `json:"current_hour"`
}

// BuildPresenceContext returns a `[presence_context: ...]` block for
// presence.enter events. Cheap: one or two file reads (today + yesterday)
// at most, no patterns / mood lookups.
//
// Returns "" when the user is unknown or empty — strangers don't carry a
// stable relational identity, so "welcome back" framing doesn't fit; the
// existing stranger-greeting path handles that case unchanged.
func BuildPresenceContext(user string) string {
	user = usercanon.Resolve(user)
	if user == "" || user == "unknown" {
		return ""
	}

	now := time.Now()
	leaveTS := wellbeing.LastActionTS(user, "leave", presenceLookbackDays)
	ageMin := -1
	if leaveTS > 0 {
		ageMin = int(now.Sub(time.Unix(int64(leaveTS), 0)).Minutes())
	}

	ctx := presenceContext{
		LastLeaveAgeMin: ageMin,
		CurrentHour:     now.Hour(),
	}
	body, err := json.Marshal(ctx)
	if err != nil {
		slog.Warn("presence context: marshal failed", "component", "skillcontext", "error", err)
		return ""
	}
	return fmt.Sprintf("\n[presence_context: %s]", string(body))
}
