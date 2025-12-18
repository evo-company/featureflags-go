package featureflags

type Conditions struct{}

func LessThan(left, right string) bool {
	return left < right
}

func Equal(left string, right string) bool {
	return left == right
}

type FlagState struct {
	Name    string
	Enabled bool
}

func (state *State) FlagState(name string) bool {
	result := false
	value, foundValue := state.flagState[name]

	if foundValue {
		result = value.Enabled
	}
	return result
}

func (flags *FeatureFlags) Get(name string) bool {
	flags.mu.RLock()
	defer flags.mu.RUnlock()
	return flags.state.FlagState(name)
}

type FlagResponse struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Flag struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}
