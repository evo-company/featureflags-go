package featureflags

import "testing"

// Test Get method for flags
func TestGet(t *testing.T) {
	flags := &FeatureFlags{
		state: State{
			flagState: map[string]FlagState{
				"enabled_flag":  {Name: "enabled_flag", Enabled: true},
				"disabled_flag": {Name: "disabled_flag", Enabled: false},
			},
		},
	}

	t.Run("get enabled flag", func(t *testing.T) {
		if !flags.Get("enabled_flag") {
			t.Error("Expected enabled_flag to be true")
		}
	})

	t.Run("get disabled flag", func(t *testing.T) {
		if flags.Get("disabled_flag") {
			t.Error("Expected disabled_flag to be false")
		}
	})

	t.Run("get non-existent flag", func(t *testing.T) {
		if flags.Get("non_existent") {
			t.Error("Expected non_existent flag to be false")
		}
	})
}
