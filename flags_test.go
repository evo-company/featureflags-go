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

// Test Get method with conditions
func TestGetWithConditions(t *testing.T) {
	// Create a flag with a condition: user.id == 123
	flags := &FeatureFlags{
		state: State{
			flagState: map[string]FlagState{
				"conditional_flag": {
					Name:    "conditional_flag",
					Enabled: false, // default is disabled
					Proc: func(ctx map[string]any) bool {
						// Simulates: enabled when user.id == 123
						if userId, ok := ctx["user.id"]; ok {
							return userId == 123
						}
						return false
					},
				},
			},
		},
	}

	t.Run("flag enabled for matching context", func(t *testing.T) {
		ctx := map[string]any{"user.id": 123}
		if !flags.Get("conditional_flag", WithContext(ctx)) {
			t.Error("Expected conditional_flag to be true for user.id=123")
		}
	})

	t.Run("flag disabled for non-matching context", func(t *testing.T) {
		ctx := map[string]any{"user.id": 456}
		if flags.Get("conditional_flag", WithContext(ctx)) {
			t.Error("Expected conditional_flag to be false for user.id=456")
		}
	})

	t.Run("flag disabled for missing context variable", func(t *testing.T) {
		ctx := map[string]any{}
		if flags.Get("conditional_flag", WithContext(ctx)) {
			t.Error("Expected conditional_flag to be false when user.id is missing")
		}
	})

	t.Run("flag disabled for nil context", func(t *testing.T) {
		if flags.Get("conditional_flag") {
			t.Error("Expected conditional_flag to be false for nil context")
		}
	})
}
