package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/lib/hal"
	"go.autonomous.ai/os/server/serializers"
)

// voicePreview plays a TTS preview through HAL using server-side
// credentials. Body: {text, voice, provider}. The TTS API key + base URL
// come from cfg (with the same LLM-fallback the runtime voice pipeline
// uses) — they never leave the device. Audit web F13: previous flow
// shipped tts_api_key in the request body straight to /hw/voice/speak.
func (s *Server) voicePreview(c *gin.Context) {
	var body struct {
		Text     string `json:"text"`
		Voice    string `json:"voice"`
		Provider string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("text required"))
		return
	}
	apiKey := s.config.GetTTSAPIKey()
	baseURL := s.config.GetTTSBaseURL()
	if err := hal.SpeakPreview(body.Text, body.Voice, body.Provider, apiKey, baseURL); err != nil {
		slog.Warn("voice preview failed", "component", "voice", "error", err)
		c.JSON(http.StatusBadGateway, serializers.ResponseError("preview failed: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}
