package skills

import "testing"

// A device with presence keeps every hook; one without it drops the expression
// hook (emotion-acknowledge) but keeps the agnostic one (turn-gate).
func TestSupportedHooks_GatesExpressionHook(t *testing.T) {
	withPresence := SupportedHooks(map[string]bool{"presence": true})
	if !contains(withPresence, "emotion-acknowledge") || !contains(withPresence, "turn-gate") {
		t.Fatalf("presence device should keep all hooks, got %v", withPresence)
	}

	noPresence := SupportedHooks(map[string]bool{"audio": true, "sensing": true})
	if contains(noPresence, "emotion-acknowledge") {
		t.Errorf("emotion-acknowledge must be pruned without presence, got %v", noPresence)
	}
	if !contains(noPresence, "turn-gate") {
		t.Errorf("turn-gate (no capability) must always survive, got %v", noPresence)
	}
}

// Empty capabilities fail open to every hook.
func TestSupportedHooks_FailOpen(t *testing.T) {
	if got := SupportedHooks(nil); len(got) != len(Hooks) {
		t.Fatalf("nil caps: got %d hooks, want all %d (fail-open)", len(got), len(Hooks))
	}
}

// Every capability the hook map references must be a real DEVICE.md capability.
func TestHookCapability_KnownCapabilities(t *testing.T) {
	known := map[string]bool{
		"audio": true, "vision": true, "sensing": true, "presence": true,
		"motion": true, "light": true, "display": true, "media": true,
		"connectivity": true, "system": true,
	}
	for hook, cap := range HookCapability {
		if !known[cap] {
			t.Errorf("hook %q maps to unknown capability %q", hook, cap)
		}
		if !contains(Hooks, hook) {
			t.Errorf("hook %q in HookCapability is not in Hooks", hook)
		}
	}
}
