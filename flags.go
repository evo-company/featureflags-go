package featureflags

// FlagState stores the state of a feature flag including its evaluation function
type FlagState struct {
	Name    string
	Enabled bool     // default value
	Proc    FlagProc // compiled condition evaluator (nil if using default)
}

// getFlagState evaluates a flag against a context and returns its state
func (state *State) getFlagState(name string, ctx map[string]any) bool {
	flagState, found := state.flagState[name]
	if !found {
		return false
	}

	// If we have a proc (conditions from server), use it
	if flagState.Proc != nil {
		return flagState.Proc(ctx)
	}

	// Otherwise return the default/static value
	return flagState.Enabled
}

// Get returns the state of a feature flag evaluated against the provided context
func (flags *FeatureFlags) Get(name string, ctx map[string]any) bool {
	flags.mu.RLock()
	defer flags.mu.RUnlock()
	return flags.state.getFlagState(name, ctx)
}

// FlagResponse represents a flag response from the server
type FlagResponse struct {
	Name       string      `json:"name"`
	Enabled    bool        `json:"enabled"`
	Overridden bool        `json:"overridden"`
	Conditions []Condition `json:"conditions"`
}

// Flag represents a flag definition with default value
type Flag struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}
