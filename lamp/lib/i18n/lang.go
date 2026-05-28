// Package i18n exposes the active STT language (Lumi config) and the
// consolidated table of short hardcoded TTS phrases — recovery
// announcements, ambient mumbles, "brain restarting" notices, etc.
// Phrase content lives in phrases.go; callers go through Pick/One
// rather than holding their own pools.
//
// The module is a singleton because the alternative — plumbing
// *config.Config through every Service constructor + Wire provider — would
// touch a lot of unrelated wiring just so an idle mumble loop can read one
// string. SetConfig is called once from server.ProvideServer / boot.
package i18n

import (
	"sync/atomic"

	"go-lamp.autonomous.ai/server/config"
)

// BCP-47 language codes used across the codebase. Defined as constants so
// switches and map keys are typo-safe at compile time and IDE jump-to-def
// surfaces every site that handles a given language. Aliases (LangZh,
// LangZhHans, LangZhHant) cover STT-config inputs we accept but normalise
// onto LangZhCN / LangZhTW for content lookups.
const (
	LangEN     = "en"
	LangVI     = "vi"
	LangZhCN   = "zh-CN"
	LangZhTW   = "zh-TW"
	LangZh     = "zh"
	LangZhHans = "zh-Hans"
	LangZhHant = "zh-Hant"
)

// active holds the Config pointer set by SetConfig. atomic.Pointer because
// SetConfig may run on a different goroutine than Lang() readers.
var active atomic.Pointer[config.Config]

// SetConfig wires the live config so Lang() returns the current setting.
// Idempotent; later calls overwrite the pointer.
func SetConfig(cfg *config.Config) {
	active.Store(cfg)
}

// Lang returns the active STT language code (e.g. "vi", "en", "zh-CN").
// Empty string when SetConfig has not been called yet — callers should
// treat empty as the English fallback.
func Lang() string {
	cfg := active.Load()
	if cfg == nil {
		return ""
	}
	return cfg.STTLanguage
}

// LangContextTag returns "\n[context: current_language=X]" when the STT
// language is configured, "" otherwise. Mirrors the [context: current_user=X]
// injection used by sensing handlers so SKILL.md replies to sensor-triggered
// (text-less) events — presence.enter/leave, motion.activity, emotion — stay
// in the right locale instead of defaulting to English. Voice and web_chat
// paths skip this because the user's own text already carries the language
// signal.
func LangContextTag() string {
	if l := Lang(); l != "" {
		return "\n[context: current_language=" + l + "]"
	}
	return ""
}
