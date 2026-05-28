package openclaw

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-lamp.autonomous.ai/lib/lelamp"
)

// --- Device identity (Ed25519) for gateway auth ---

const deviceKeyFile = "lumi-device-key.json"

type deviceIdentity struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	DeviceID   string // hex(SHA-256(publicKey))
}

// loadOrCreateDeviceIdentity loads the Ed25519 keypair from disk, or generates
// a new one and persists it for future connections.
func (s *Service) loadOrCreateDeviceIdentity() (*deviceIdentity, error) {
	keyPath := filepath.Join(s.config.OpenclawConfigDir, deviceKeyFile)
	if data, err := os.ReadFile(keyPath); err == nil {
		var stored struct {
			PrivateKey string `json:"privateKey"` // hex-encoded 64-byte Ed25519 seed+pub
		}
		if err := json.Unmarshal(data, &stored); err == nil {
			privBytes, err := hex.DecodeString(stored.PrivateKey)
			if err == nil && len(privBytes) == ed25519.PrivateKeySize {
				priv := ed25519.PrivateKey(privBytes)
				pub := priv.Public().(ed25519.PublicKey)
				id := deriveDeviceID(pub)
				slog.Info("loaded device identity", "component", "openclaw", "deviceId", id)
				return &deviceIdentity{PublicKey: pub, PrivateKey: priv, DeviceID: id}, nil
			}
		}
	}

	// Generate new keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	id := deriveDeviceID(pub)

	stored := map[string]string{"privateKey": hex.EncodeToString(priv)}
	data, _ := json.MarshalIndent(stored, "", "  ")
	if err := os.WriteFile(keyPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write device key: %w", err)
	}
	_ = chownRuntimeUserIfRoot(keyPath, openclawRuntimeUser)
	slog.Info("generated new device identity", "component", "openclaw", "deviceId", id)
	return &deviceIdentity{PublicKey: pub, PrivateKey: priv, DeviceID: id}, nil
}

// deriveDeviceID returns hex(SHA-256(rawPublicKey)).
func deriveDeviceID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:])
}

// signConnectPayload builds and signs the v2 payload for device auth.
// Format: v2|deviceId|clientId|clientMode|role|scopes|signedAtMs|token|nonce
func (di *deviceIdentity) signConnectPayload(token, nonce string, signedAt int64) string {
	payload := fmt.Sprintf("v2|%s|%s|%s|%s|%s|%d|%s|%s",
		di.DeviceID,
		"node-host", // clientId
		"node",      // clientMode
		"operator",  // role
		"operator.read,operator.write,operator.admin", // scopes
		signedAt,
		token,
		nonce,
	)
	sig := ed25519.Sign(di.PrivateKey, []byte(payload))
	return base64.StdEncoding.EncodeToString(sig)
}

// WatchIdentity polls IDENTITY.md in the OpenClaw workspace and pushes updated wake words
// to LeLamp whenever the agent's name changes (e.g. user says "call yourself Noah").
func (s *Service) WatchIdentity(ctx context.Context) {
	identityPath := filepath.Join(s.config.OpenclawConfigDir, "workspace", "IDENTITY.md")
	var lastName string
	for {
		if !sleepCtx(ctx, 5*time.Second) {
			return
		}
		data, err := os.ReadFile(identityPath)
		if err != nil {
			continue
		}
		name := parseIdentityName(string(data))
		if name == "" || name == lastName {
			continue
		}
		lastName = name
		words := buildWakeWords(name)
		slog.Info("agent renamed, updating wake words", "component", "openclaw", "name", name, "words", words)
		lelamp.SetVoiceConfig(words)
	}
}

// parseIdentityName extracts the agent name from IDENTITY.md content.
// Looks for a line matching: - **Name:** <value>
func parseIdentityName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Match: - **Name:** Lumi  or  **Name:** Lumi
		lower := strings.ToLower(line)
		idx := strings.Index(lower, "**name:**")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[idx+len("**name:**"):])
		// Strip trailing markdown (e.g. " — some description")
		if i := strings.IndexAny(name, "—-|"); i > 0 {
			name = strings.TrimSpace(name[:i])
		}
		if name != "" {
			return name
		}
	}
	return ""
}

// buildWakeWords generates wake word variants from an agent name.
func buildWakeWords(name string) []string {
	n := strings.ToLower(name)
	return []string{
		"hey " + n,
		n,
		"này " + n,
		"ê " + n,
		n + " ơi",
	}
}
