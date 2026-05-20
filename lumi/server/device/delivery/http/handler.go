package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go-lamp.autonomous.ai/domain"
	"go-lamp.autonomous.ai/internal/device"
	"go-lamp.autonomous.ai/internal/network"
	"go-lamp.autonomous.ai/lib/lelamp"
	"go-lamp.autonomous.ai/server/config"
	"go-lamp.autonomous.ai/server/serializers"
	"go-lamp.autonomous.ai/server/session"
)

// DeviceHandler represents the HTTP handler for device
type DeviceHandler struct {
	service        *device.Service
	networkService *network.Service
	config         *config.Config
}

func ProvideDeviceHandler(ds *device.Service, ns *network.Service, cfg *config.Config) DeviceHandler {
	return DeviceHandler{
		service:        ds,
		networkService: ns,
		config:         cfg,
	}
}

// Setup godoc
//
//	@Summary	setup device
//	@Schemes
//	@Description	setup device
//	@Tags			device
//	@Accept			json
//	@Param			body	body		domain.SetupRequest		true	"setup request"
//	@Success		200		{object}	serializers.ResponseSuccess
//	@Router			/device/setup [post]
func (h *DeviceHandler) Setup(c *gin.Context) {
	var req domain.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := validator.New().Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := req.ValidateChannel(); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}

	// If operator supplied an admin password, set the session cookie now so the
	// browser is logged in by the time it redirects post-setup. The hash itself
	// is persisted asynchronously inside service.Setup; the cookie validates
	// against SessionSecret (independent of the password hash), so there's no
	// race — any subsequent /api/* call sees a valid session immediately.
	if req.AdminPassword != "" {
		if err := session.Issue(c, h.config); err != nil {
			slog.Warn("setup: issue session failed", "component", "device", "error", err)
		}
	}

	go func() {
		time.Sleep(2 * time.Second)
		if err := h.service.Setup(req); err != nil {
			slog.Error("setup failed", "component", "device", "error", err)
			h.networkService.SwitchToAPMode()
			return
		}

		slog.Info("setup success", "component", "device")
	}()

	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// GetConfig godoc
//
//	@Summary	get current device config (sanitized)
//	@Schemes
//	@Description	get current device config. Secrets (API keys, channel
//	@Description	tokens, passwords) are returned as Has* booleans only —
//	@Description	plaintext values never leave the device. Use PUT
//	@Description	/api/device/config to update individual secret fields.
//	@Tags			device
//	@Success		200	{object}	serializers.ResponseSuccess
//	@Router			/device/config [get]
func (h *DeviceHandler) GetConfig(c *gin.Context) {
	cfg := h.service.GetPublicConfig()
	c.JSON(http.StatusOK, serializers.ResponseSuccess(cfg))
}

// SetupStatus godoc
//
//	@Summary	current setup phase + LAN IP
//	@Description	web polls this during the AP→STA transition to learn the
//	@Description	device's new LAN IP and redirect the user. Phase progresses
//	@Description	idle → connecting → connected (or failed).
//	@Tags			device
//	@Success		200	{object}	serializers.ResponseSuccess
//	@Router			/device/setup/status [get]
func (h *DeviceHandler) SetupStatus(c *gin.Context) {
	phase, lanIP, errMsg := h.service.SetupStatus()
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
		"phase":  phase,
		"lan_ip": lanIP,
		"error":  errMsg,
	}))
}

// UpdateConfig godoc
//
//	@Summary	update device config
//	@Schemes
//	@Description	update device config fields (all optional; saves to disk, restart Lumi for full effect)
//	@Tags			device
//	@Accept			json
//	@Param			body	body		domain.UpdateConfigRequest	true	"update config request"
//	@Success		200		{object}	serializers.ResponseSuccess
//	@Router			/device/config [put]
func (h *DeviceHandler) UpdateConfig(c *gin.Context) {
	var req domain.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.UpdateConfig(req); err != nil {
		c.JSON(http.StatusInternalServerError, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// GetVoices returns the list of available TTS voices for the requested provider.
// Tries LeLamp /voice/voices?provider=&lang= first (source of truth), falls
// back to a static list. `lang` (BCP-47 stt_language code) lets the web UI
// filter voices to those that sound natural in the active language; empty
// lang returns the full flat list.
func (h *DeviceHandler) GetVoices(c *gin.Context) {
	provider := c.DefaultQuery("provider", domain.TTSProviderOpenAI)
	lang := c.Query("lang")

	if voices, err := lelamp.ListVoices(provider, lang); err == nil && len(voices) > 0 {
		c.JSON(http.StatusOK, serializers.ResponseSuccess(voices))
		return
	}
	// Fallback to static list (no language filtering — static list is EN-only)
	voices, ok := domain.TTSVoicesByProvider[provider]
	if !ok {
		voices = domain.TTSVoices
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(voices))
}

// GetTTSProviders returns the list of supported TTS providers.
func (h *DeviceHandler) GetTTSProviders(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(domain.TTSProviders))
}

// ChangeChannel godoc
//
//	@Summary	change messaging channel
//	@Schemes
//	@Description	change messaging channel (telegram/slack/discord) without full device re-setup
//	@Tags			device
//	@Accept			json
//	@Param			body	body		domain.ChangeChannelRequest	true	"change channel request"
//	@Success		200		{object}	serializers.ResponseSuccess
//	@Router			/device/channel [post]
func (h *DeviceHandler) ChangeChannel(c *gin.Context) {
	var req domain.AddChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := validator.New().Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := req.ValidateChannel(); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}

	go func() {
		if err := h.service.AddChannel(req); err != nil {
			slog.Error("add channel failed", "component", "device", "error", err)
			return
		}
		slog.Info("add channel success", "component", "device")
	}()

	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}
