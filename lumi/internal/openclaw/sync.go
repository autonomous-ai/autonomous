package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-lamp.autonomous.ai/domain"
)

// SyncModelsFromAPI fetches the live model list from ModelsAPIURL and
// reconciles it into openclaw.json under s.config.OpenclawConfigDir. The
// merge is additive:
//   - providers.autonomous.models[] gains entries for any new model id;
//     nothing is ever removed.
//   - agents.defaults.models gains a "{provider}/{key}" entry for each model
//     in the API response. Legacy unprefixed keys for the same model are
//     replaced with the prefixed form (metadata preserved). Keys for models
//     not in the API response are left untouched.
//
// No-op (returns false, nil) when openclaw.json is missing or the provider
// section is absent. A failed fetch / invalid JSON returns an error so the
// caller can decide to fall back. The caller must NOT treat any of these as
// fatal — the device must keep running.
//
// Restarts the openclaw gateway only when the file actually changed.
// Holds primarySyncMu for the entire read-modify-write cycle so it cannot
// interleave with other openclaw.json writers (watcher, refresh, setup).
// The network fetch happens before the lock to keep the critical section short.
func (s *Service) SyncModelsFromAPI() (bool, error) {
	resp, err := FetchModelsFromAPI()
	if err != nil {
		return false, fmt.Errorf("fetch models: %w", err)
	}

	s.primarySyncMu.Lock()
	defer s.primarySyncMu.Unlock()

	configPath := filepath.Join(s.config.OpenclawConfigDir, "openclaw.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read openclaw config: %w", err)
	}
	var configData map[string]interface{}
	if err := json.Unmarshal(raw, &configData); err != nil {
		return false, fmt.Errorf("parse openclaw config: %w", err)
	}

	autonomousMap, ok := autonomousProviderMap(configData)
	if !ok {
		return false, nil
	}

	return applyModelsToConfig(configPath, configData, autonomousMap, resp.Models)
}

// StartModelSync runs the periodic model sync loop until ctx is cancelled.
// Eager first tick on entry, then a steady ticker at ModelSyncInterval. Each
// tick is wrapped in panic recovery so a third-party JSON parser regression
// can't kill the loop. A failed sync logs and continues — the device must
// keep running.
func (s *Service) StartModelSync(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[modelsync] PANIC recovered, sync loop stopped", "panic", r)
		}
	}()

	tick := func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("[modelsync] tick PANIC recovered", "panic", r)
			}
		}()
		if _, err := s.SyncModelsFromAPI(); err != nil {
			slog.Warn("[modelsync] tick failed", "err", err)
		}
	}

	tick()
	ticker := time.NewTicker(ModelSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

// FetchModelsFromAPI does the actual HTTP GET against ModelsAPIURL (tunables.go)
// and returns the upstream model list. Used by the periodic SyncModelsFromAPI
// loop. Returns a typed error on transport, status, or JSON-shape failures so
// callers can skip the tick without crashing the device.
func FetchModelsFromAPI() (*domain.LLMModelsListResponse, error) {
	url := strings.TrimSpace(ModelsAPIURL)
	if url == "" {
		return nil, fmt.Errorf("empty models api url (check tunables.go)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), modelsAPITimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build models request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	var out domain.LLMModelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}
	if len(out.Models) == 0 {
		return nil, fmt.Errorf("models response is empty")
	}
	return &out, nil
}

// autonomousProviderMap drills into models.providers.autonomous, returning the
// inner map and ok=true only when every level exists.
func autonomousProviderMap(configData map[string]any) (map[string]any, bool) {
	modelsMap, _ := configData["models"].(map[string]any)
	if modelsMap == nil {
		return nil, false
	}
	providersMap, _ := modelsMap["providers"].(map[string]any)
	if providersMap == nil {
		return nil, false
	}
	autonomousMap, _ := providersMap[customProviderName].(map[string]any)
	if autonomousMap == nil {
		return nil, false
	}
	return autonomousMap, true
}

// applyModelsToConfig merges the given fetched models into both
// providers.autonomous.models and agents.defaults.models, writes the file when
// anything changed, and restarts the openclaw gateway. Idempotent.
func applyModelsToConfig(configPath string, configData map[string]any, autonomousMap map[string]any, fetched []domain.LLMModel) (bool, error) {
	existingProvider, _ := autonomousMap["models"].([]any)
	mergedProvider, providerChanged := mergeProviderModels(existingProvider, fetched)

	var agentChanged bool
	if agentsMap, ok := configData["agents"].(map[string]any); ok {
		if defaultsMap, ok := agentsMap["defaults"].(map[string]any); ok {
			existingAgent, _ := defaultsMap["models"].(map[string]any)
			merged, changed := mergeAgentModels(existingAgent, fetched)
			if changed {
				defaultsMap["models"] = merged
				agentChanged = true
			}
		}
	}

	if !providerChanged && !agentChanged {
		return false, nil
	}
	if providerChanged {
		autonomousMap["models"] = mergedProvider
	}

	written, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal openclaw config: %w", err)
	}
	// Write the current primary into the flag so the watcher can match by
	// content: model-list sync never changes primary, so the flag value equals
	// whatever is already set, and the watcher correctly skips this write.
	setLumiWriteFlag(filepath.Dir(configPath), extractPrimaryModel(configData))
	if err := atomicWriteFile(configPath, written, 0600); err != nil {
		return false, fmt.Errorf("write openclaw config: %w", err)
	}
	if err := chownRuntimeUserIfRoot(configPath, openclawRuntimeUser); err != nil {
		return false, fmt.Errorf("set openclaw config ownership: %w", err)
	}
	slog.Info("[modelsync] reconciled openclaw config",
		"path", configPath,
		"provider_changed", providerChanged,
		"agent_changed", agentChanged,
		"fetched", len(fetched),
	)

	if err := restartOpenclawGateway(); err != nil {
		slog.Warn("[modelsync] restart openclaw gateway", "err", err)
	}
	return true, nil
}

// mergeProviderModels reconciles the providers.autonomous.models[] slice with
// the fetched list:
//   - Models in fetched whose id is missing from existing are APPENDED.
//   - Models present in both have their contextWindow and maxTokens REFRESHED
//     to the upstream values (other fields are left as-is to preserve any
//     local edits a reviewer may have made).
//   - Models in existing but not in fetched are KEPT untouched (additive
//     reconciliation — never removes).
//
// Returns the (possibly modified) slice and a bool indicating whether anything
// was added or updated.
func mergeProviderModels(existing []any, fetched []domain.LLMModel) ([]any, bool) {
	indexByID := make(map[string]int, len(existing))
	for i, e := range existing {
		if m, ok := e.(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				indexByID[id] = i
			}
		}
	}
	out := append([]any(nil), existing...)
	changed := false
	for _, m := range fetched {
		fresh := openclawModelToProviderEntry(m)
		idx, ok := indexByID[m.Key]
		if !ok {
			out = append(out, fresh)
			indexByID[m.Key] = len(out) - 1
			changed = true
			continue
		}
		entry, ok := out[idx].(map[string]any)
		if !ok {
			continue
		}
		if !numbersEqual(entry["contextWindow"], fresh["contextWindow"]) {
			entry["contextWindow"] = fresh["contextWindow"]
			changed = true
		}
		if !numbersEqual(entry["maxTokens"], fresh["maxTokens"]) {
			entry["maxTokens"] = fresh["maxTokens"]
			changed = true
		}
		out[idx] = entry
	}
	return out, changed
}

// numbersEqual compares two JSON-decoded numeric values that may be int,
// int64, or float64 depending on whether they came from a Go literal or from
// json.Unmarshal into map[string]any. Returns false if either side is not a
// number.
func numbersEqual(a, b any) bool {
	av, aOk := toFloat(a)
	bv, bOk := toFloat(b)
	if !aOk || !bOk {
		return false
	}
	return av == bv
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

// mergeAgentModels reconciles the agents.defaults.models map with fetched
// models. For every fetched model:
//   - If "{provider}/{key}" exists, keep its metadata.
//   - Otherwise, if the legacy unprefixed "{key}" exists, migrate its metadata
//     to the prefixed form and drop the legacy entry.
//   - Otherwise, add a new prefixed entry with empty metadata.
//
// Keys not corresponding to any fetched model are preserved as-is.
func mergeAgentModels(existing map[string]any, fetched []domain.LLMModel) (map[string]any, bool) {
	out := make(map[string]any, len(existing)+len(fetched))
	for k, v := range existing {
		out[k] = v
	}
	changed := false
	for _, m := range fetched {
		newKey := agentModelKey(m)
		if _, ok := out[newKey]; ok {
			continue
		}
		if legacy, ok := out[m.Key]; ok {
			out[newKey] = legacy
			delete(out, m.Key)
			changed = true
			continue
		}
		out[newKey] = map[string]any{}
		changed = true
	}
	return out, changed
}

// agentModelKey returns the key used under agents.defaults.models for a given
// provider model. The "{provider}/{key}" shape keeps the openclaw gateway's
// /models listing grouped under a single provider (otherwise it splits ids
// like "minimax/minimax-m2.7" on the first slash and shows them as separate
// providers).
func agentModelKey(m domain.LLMModel) string {
	return customProviderName + "/" + m.Key
}

// atomicWriteFile writes data to path so that concurrent readers — and most
// importantly an unexpected power loss between the open-truncate and the
// final write — never observe a half-written file. Implemented via the
// standard write-temp-then-rename pattern: rename(2) on POSIX is guaranteed
// to either expose the new file in full or leave the old file intact. If the
// process is killed mid-write, the temp file is left behind harmlessly and
// the next sync tick will overwrite it.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".openclaw-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
