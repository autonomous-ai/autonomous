package hermes

import (
	"fmt"
	"log/slog"
	"time"
)

// rotateMaxTurns / rotateTokenThreshold gate conversation rotation (see
// ShouldRotateSession). The generic handler's autoSessionThreshold (150k tokens)
// is useless for Hermes: the gateway compresses history server-side before each
// call, so os-server only ever observes ~20-60k tokens regardless of the real
// chain size — which grows to millions of tokens / tens of MB per device-main
// response blob and makes every turn reconstruct + recompress it (~1min/turn).
const (
	rotateMaxTurns       = 40
	rotateTokenThreshold = 50_000
)

// initConversation seeds the active conversation name once per process with a
// boot-unique suffix, so a restart never re-attaches to a previously bloated
// chain (the gateway keys its response history on the conversation name).
func (s *HermesService) initConversation() {
	s.convOnce.Do(func() {
		s.bootStamp = time.Now().Unix()
		s.conversation.Store(fmt.Sprintf("%s-%d", Conversation, s.bootStamp))
	})
}

// conversationName returns the active conversation name sent on every turn.
func (s *HermesService) conversationName() string {
	s.initConversation()
	name, _ := s.conversation.Load().(string)
	return name
}

// rotateConversation switches future turns to a fresh conversation name so the
// gateway starts a new (small) history chain. The old chain is abandoned (it
// remains on the gateway under the old name until a separate prune reclaims the
// disk). Clears lastResponseID so os-server stops correlating the old chain.
func (s *HermesService) rotateConversation() {
	s.initConversation()
	seq := s.rotateSeq.Add(1)
	name := fmt.Sprintf("%s-%d-%d", Conversation, s.bootStamp, seq)
	s.conversation.Store(name)
	s.lastResponseID.Store("")
	slog.Info("hermes conversation rotated", "component", "hermes", "conversation", name)
}

// ShouldRotateSession overrides the generic handler's token-threshold rotation
// decision (the sessionRotator optional interface). Hermes rotates on turn count
// (primary — the gateway blob grows ~one history snapshot per turn) or a token
// spike (secondary), because the generic 150k-token trigger never fires: the
// gateway compresses history before each call so os-server only observes
// ~20-60k tokens, never the real multi-million-token size.
func (s *HermesService) ShouldRotateSession(totalTokens, turnsSinceRotation int) bool {
	return turnsSinceRotation >= rotateMaxTurns || totalTokens >= rotateTokenThreshold
}

// NewSession rotates the conversation. Both the generic handler's auto-new-session
// path and an explicit factory reset land here. Instant — no gateway RPC.
func (s *HermesService) NewSession(sessionKey string) error {
	slog.Info("hermes NewSession: rotating conversation", "component", "hermes", "key", sessionKey)
	s.rotateConversation()
	return nil
}
