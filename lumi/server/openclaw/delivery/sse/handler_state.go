package sse

import (
	"log/slog"
	"strings"
)

// accumulateAssistantDelta appends a delta to the buffer for the given runId.
func (h *OpenClawHandler) accumulateAssistantDelta(runID, delta string) {
	if delta == "" {
		return
	}
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	buf, ok := h.assistantBuf[runID]
	if !ok {
		buf = &strings.Builder{}
		h.assistantBuf[runID] = buf
	}
	buf.WriteString(delta)
	slog.Info("assistant delta buffered (TTS waits for lifecycle:end)",
		"component", "agent",
		"run_id", runID,
		"delta", delta,
		"cumulative_len", buf.Len(),
		"cumulative_tail", tailPreview(buf.String(), 120),
	)
}

// tailPreview returns the last n chars of s for log readability without spamming
// the entire growing buffer on every delta.
func tailPreview(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// tryFirstSentenceFlush returns the FIRST complete sentence in the per-run
// buffer once it is safe to stream to TTS, or "" when no sentence is ready
// or one has already been streamed for this run. The raw buffer is left
// intact so flushAssistantText at lifecycle:end still sees every HW marker.
//
// Why only the first sentence: chaining every sentence as its own
// /voice/speak POST exposes a ~400ms ElevenLabs/OpenAI TTFB gap between
// each one — perceived as choppy. Streaming just the first sentence wins
// most of the first-audio latency (~2s) while letting the remainder go
// through /voice/speak-queue at lifecycle:end, which Python pre-synthesises
// while the first sentence is still playing.
//
// Defers (returns "") when the snapshot has:
//   - a partial `[HW:` marker (extractHWCalls only matches complete markers)
//   - any `<say>` wrapper (extractSayTag at end shifts content)
//   - `NO_REPLY` / `HEARTBEAT_OK` sentinels (sanitizeAgentText strips
//     these at end-flush; streamed text can't be unspoken)
func (h *OpenClawHandler) tryFirstSentenceFlush(runID string) string {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()

	// Already streamed first sentence for this run — let lifecycle:end
	// handle the rest via /voice/speak-queue.
	if _, already := h.streamedCleanLen[runID]; already {
		return ""
	}
	buf, ok := h.assistantBuf[runID]
	if !ok || buf.Len() == 0 {
		return ""
	}
	raw := buf.String()
	if hasPartialHWMarker(raw) {
		return ""
	}
	if strings.Contains(raw, "<say>") {
		return ""
	}
	upper := strings.ToUpper(raw)
	if strings.Contains(upper, "NO_REPLY") || strings.Contains(upper, "HEARTBEAT_OK") {
		return ""
	}

	_, cleaned := extractHWCalls(raw)
	cleaned = prunedImageMarkerRe.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}

	boundary := findSentenceFlushBoundary(cleaned)
	if boundary < 0 {
		return ""
	}
	sentence := strings.TrimSpace(cleaned[:boundary+1])
	if sentence == "" {
		return ""
	}
	h.streamedCleanLen[runID] = boundary + 1
	return sentence
}

// consumeStreamedCleanLen returns the byte offset into the cleaned reply
// already streamed to TTS for runID and clears the entry. Called at
// lifecycle:end so the remainder POST sends only what was not already
// streamed. Returns 0 when no sentence was streamed for this run.
func (h *OpenClawHandler) consumeStreamedCleanLen(runID string) int {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	n, ok := h.streamedCleanLen[runID]
	if !ok {
		return 0
	}
	delete(h.streamedCleanLen, runID)
	return n
}

// hasPartialHWMarker reports whether text contains a `[HW:` opener with no
// matching `]` before EOF. extractHWCalls only matches complete markers, so
// a partial marker would survive into cleaned text and could be split
// mid-sentence. tryFirstSentenceFlush defers in that case.
func hasPartialHWMarker(text string) bool {
	idx := strings.Index(text, "[HW:")
	for idx >= 0 {
		end := strings.Index(text[idx:], "]")
		if end < 0 {
			return true
		}
		next := strings.Index(text[idx+4:], "[HW:")
		if next < 0 {
			return false
		}
		idx = idx + 4 + next
	}
	return false
}

// findSentenceFlushBoundary returns the rightmost index in s of `[.?!]`
// followed by whitespace, or -1 if none. The trailing-whitespace requirement
// confirms the next token has begun (so we're not splitting an abbreviation
// or version number mid-formation). Decimal patterns "5. 5" are also skipped.
func findSentenceFlushBoundary(s string) int {
	n := len(s)
	for i := n - 2; i >= 0; i-- {
		c := s[i]
		if c != '.' && c != '?' && c != '!' {
			continue
		}
		next := s[i+1]
		if next != ' ' && next != '\n' && next != '\t' && next != '\r' {
			continue
		}
		if i > 0 && isAsciiDigit(s[i-1]) {
			j := i + 1
			for j < n && (s[j] == ' ' || s[j] == '\t') {
				j++
			}
			if j < n && isAsciiDigit(s[j]) {
				continue
			}
		}
		return i
	}
	return -1
}

func isAsciiDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// flushAssistantText returns the accumulated text for runId and clears the buffer.
// HW markers are stripped here so they never appear in Telegram or other channel replies.
// The caller is responsible for extracting and firing HW calls before flushing.
func (h *OpenClawHandler) flushAssistantText(runID string) (string, []hwCall) {
	h.assistantMu.Lock()
	defer h.assistantMu.Unlock()
	buf, ok := h.assistantBuf[runID]
	if !ok || buf.Len() == 0 {
		return "", nil
	}
	raw := buf.String()
	raw = prunedImageMarkerRe.ReplaceAllString(raw, "")
	calls, text := extractHWCalls(raw)
	text = strings.TrimSpace(text)
	delete(h.assistantBuf, runID)
	return text, calls
}

// suppressTTS flags a runID to skip TTS on lifecycle end with the given reason.
func (h *OpenClawHandler) suppressTTS(runID, reason string) {
	h.ttsSuppressMu.Lock()
	defer h.ttsSuppressMu.Unlock()
	// "music_playing" takes priority over "already_spoken" (speaker conflict is more important).
	if existing := h.ttsSuppressReasons[runID]; existing == "music_playing" && reason != "music_playing" {
		return
	}
	h.ttsSuppressReasons[runID] = reason
}

// clearTTSSuppress removes the suppress flag for a runID and returns the reason (empty if none).
func (h *OpenClawHandler) clearTTSSuppress(runID string) string {
	h.ttsSuppressMu.Lock()
	defer h.ttsSuppressMu.Unlock()
	reason := h.ttsSuppressReasons[runID]
	delete(h.ttsSuppressReasons, runID)
	return reason
}

// resolveRunID maps an OpenClaw-assigned UUID back to the device idempotencyKey if known.
// If no mapping exists, returns the original runID unchanged.
func (h *OpenClawHandler) resolveRunID(runID string) string {
	h.runIDMapMu.Lock()
	defer h.runIDMapMu.Unlock()
	if mapped, ok := h.runIDMap[runID]; ok {
		return mapped
	}
	return runID
}

// mapRunID records that OpenClaw UUID corresponds to the given device trace (idempotencyKey).
func (h *OpenClawHandler) mapRunID(openclawID, deviceID string) {
	h.runIDMapMu.Lock()
	defer h.runIDMapMu.Unlock()
	h.runIDMap[openclawID] = deviceID
	// Limit map size to prevent unbounded growth
	if len(h.runIDMap) > 200 {
		for k := range h.runIDMap {
			delete(h.runIDMap, k)
			break
		}
	}
}
