package openclaw

import "strings"

// setdefault.go holds the pure helpers that apply the upstream-recommended
// default text model and default image model onto an already-parsed
// openclaw.json map. They operate on the in-memory map (not the file) so the
// caller — SetupAgent (first write) and SyncModelsFromAPI (periodic) — can fold
// the catalog overwrite and the default-model writes into a single
// read-modify-write + one gateway restart.
//
// Each apply* returns true only when it actually changed the map. Each is*
// gate reports whether the corresponding field is still owned by the
// "autonomous" provider (empty/missing counts as owned, i.e. safe to seed), so
// a user who manually switched a model to another provider keeps their choice.

// applyDefaultPrimaryModel sets agents.defaults.model.primary to
// "autonomous/<model>". No-op (returns false) when model is empty or already set.
func applyDefaultPrimaryModel(configData map[string]any, model string) bool {
	if model == "" {
		return false
	}
	target := customProviderName + "/" + model
	if extractPrimaryModel(configData) == target {
		return false
	}
	agents := ensureMap(configData, "agents")
	defaults := ensureMap(agents, "defaults")
	modelMap := ensureMap(defaults, "model")
	modelMap["primary"] = target
	defaults["model"] = modelMap
	agents["defaults"] = defaults
	configData["agents"] = agents
	return true
}

// applyDefaultImageModel sets agents.defaults.imageModel.primary to
// "autonomous/<imageModel>". No-op when imageModel is empty or already set.
func applyDefaultImageModel(configData map[string]any, imageModel string) bool {
	if imageModel == "" {
		return false
	}
	target := customProviderName + "/" + imageModel
	if extractImageModel(configData) == target {
		return false
	}
	agents := ensureMap(configData, "agents")
	defaults := ensureMap(agents, "defaults")
	imageMap := ensureMap(defaults, "imageModel")
	imageMap["primary"] = target
	defaults["imageModel"] = imageMap
	agents["defaults"] = defaults
	configData["agents"] = agents
	return true
}

// extractImageModel mirrors extractPrimaryModel (watch_primary.go) for
// agents.defaults.imageModel.primary. Returns "" when any level is absent.
func extractImageModel(cfg map[string]any) string {
	agents, _ := cfg["agents"].(map[string]any)
	if agents == nil {
		return ""
	}
	defaults, _ := agents["defaults"].(map[string]any)
	if defaults == nil {
		return ""
	}
	image, _ := defaults["imageModel"].(map[string]any)
	if image == nil {
		return ""
	}
	primary, _ := image["primary"].(string)
	return primary
}

// isDefaultsOnAutonomous reports whether agents.defaults.model.primary is
// empty/missing OR carries the "autonomous/" prefix — i.e. Lamp may seed it
// from upstream. Returns false when the user switched the primary to another
// provider, in which case the caller must preserve their choice.
func isDefaultsOnAutonomous(configData map[string]any) bool {
	primary := extractPrimaryModel(configData)
	if primary == "" {
		return true
	}
	return strings.HasPrefix(primary, customProviderName+"/")
}

// isImageModelOnAutonomous mirrors isDefaultsOnAutonomous for the image model,
// gated INDEPENDENTLY so a user who switched only their text model still gets
// the upstream default image model seeded.
func isImageModelOnAutonomous(configData map[string]any) bool {
	primary := extractImageModel(configData)
	if primary == "" {
		return true
	}
	return strings.HasPrefix(primary, customProviderName+"/")
}
