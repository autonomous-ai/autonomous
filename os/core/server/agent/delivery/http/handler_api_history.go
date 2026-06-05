package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/lib/mood"
	"go.autonomous.ai/os/lib/musicsuggestion"
	"go.autonomous.ai/os/lib/usercanon"
	"go.autonomous.ai/os/lib/posture"
	"go.autonomous.ai/os/lib/wellbeing"
	"go.autonomous.ai/os/server/serializers"
)

func (h *AgentHandler) MoodHistory(c *gin.Context) {
	user := usercanon.Resolve(c.DefaultQuery("user", mood.CurrentUser()))
	day := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	last := 100
	if s := c.Query("last"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			last = n
		}
	}
	if last > 500 {
		last = 500
	}
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	if kind != "" && kind != mood.KindSignal && kind != mood.KindDecision {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("kind must be signal, decision, or empty"))
		return
	}
	events := mood.Query(user, day, kind, last)
	if events == nil {
		events = []mood.Event{}
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(map[string]any{
		"date":   day,
		"kind":   kind,
		"events": events,
	}))
}

// WellbeingHistory returns wellbeing activity events for a user and day.
// Query params: user=<name> (default: current user), date=YYYY-MM-DD (default today), last=<n> (default 100, max 500).
func (h *AgentHandler) WellbeingHistory(c *gin.Context) {
	user := usercanon.Resolve(c.DefaultQuery("user", mood.CurrentUser()))
	day := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	last := 100
	if s := c.Query("last"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			last = n
		}
	}
	if last > 500 {
		last = 500
	}
	events := wellbeing.Query(user, day, last)
	if events == nil {
		events = []wellbeing.Event{}
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(map[string]any{
		"user":   user,
		"date":   day,
		"events": events,
	}))
}

// PostureHistory returns posture coach events for a user and day.
// Query params: user=<name> (default: current user), date=YYYY-MM-DD (default today), last=<n> (default 100, max 500).
func (h *AgentHandler) PostureHistory(c *gin.Context) {
	user := usercanon.Resolve(c.DefaultQuery("user", mood.CurrentUser()))
	day := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	last := 100
	if s := c.Query("last"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			last = n
		}
	}
	if last > 500 {
		last = 500
	}
	events := posture.Query(user, day, last)
	if events == nil {
		events = []posture.Event{}
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(map[string]any{
		"user":   user,
		"date":   day,
		"events": events,
	}))
}

// MusicSuggestionHistory returns music suggestion events for a user.
func (h *AgentHandler) MusicSuggestionHistory(c *gin.Context) {
	user := usercanon.Resolve(c.DefaultQuery("user", mood.CurrentUser()))
	day := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	last := 100
	if s := c.Query("last"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			last = n
		}
	}
	if last > 500 {
		last = 500
	}
	events := musicsuggestion.Query(user, day, last)
	if events == nil {
		events = []musicsuggestion.Event{}
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(map[string]any{
		"date":   day,
		"events": events,
	}))
}

// FlowStream streams today's flow events when the JSONL file changes.
