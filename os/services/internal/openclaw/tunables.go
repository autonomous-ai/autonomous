package openclaw

import "time"

// ============================================================================
// Production tunables — edit these before `make build-lamp` to change
// runtime behavior. Compile-time constants (no env var, no JSON config) so
// they are easy to grep and audit.
// ============================================================================

// ModelsAPIURL is the FULL upstream URL each device fetches to list available
// LLM models. It is intentionally hardcoded here (not read from openclaw.json)
// so the source of truth lives next to ModelSyncInterval — one file to edit
// when switching environments before `make build-lamp`.
//
// Production:
//
//	https://campaign-api.autonomous.ai/api/v1/ai/v1/models
//
// Staging:
//
//	https://campaign-api.staging.autonomousdev.xyz/api/v1/ai/v1/models
const ModelsAPIURL = "https://campaign-api.autonomous.ai/api/v1/ai/v1/models"

// ModelSyncInterval is how often each device re-fetches ModelsAPIURL and
// reconciles the result into openclaw.json under s.config.OpenclawConfigDir.
//
// The fetch is fail-soft: a missing endpoint, 5xx response, or invalid JSON
// just logs `[modelsync] tick failed: ...` and leaves the file untouched.
// While the API is not yet live in production, the loop will log one error
// per tick — bump this to a longer interval (e.g. 1*time.Hour) to reduce log
// noise until the endpoint ships, then drop it back to a shorter value once
// you want models to propagate quickly.
const ModelSyncInterval = 30 * time.Minute

// modelsAPITimeout caps a single upstream fetch so the sync loop never blocks
// forever on a hung connection.
const modelsAPITimeout = 15 * time.Second
