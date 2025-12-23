package featureflags

import "testing"

// Test GetValue method
func TestGetValue(t *testing.T) {
	flags := &FeatureFlags{
		state: State{
			valueState: map[string]ValueState{
				"string_value": {Name: "string_value", Value: "hello", DefaultValue: "default", IsOverridden: true},
				"int_value":    {Name: "int_value", Value: 42, DefaultValue: 10, IsOverridden: true},
				"float_value":  {Name: "float_value", Value: 3.14, DefaultValue: 0.0, IsOverridden: false},
			},
		},
	}

	t.Run("get string value", func(t *testing.T) {
		val := flags.GetValue("string_value", nil)
		if strVal, ok := val.(string); !ok || strVal != "hello" {
			t.Errorf("Expected 'hello', got %v", val)
		}
	})

	t.Run("get int value", func(t *testing.T) {
		val := flags.GetValue("int_value", nil)
		if intVal, ok := val.(int); !ok || intVal != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})

	t.Run("get non-existent value", func(t *testing.T) {
		val := flags.GetValue("non_existent", nil)
		if val != nil {
			t.Errorf("Expected nil, got %v", val)
		}
	})
}

// Test GetValue with conditions
func TestGetValueWithConditions(t *testing.T) {
	flags := &FeatureFlags{
		state: State{
			valueState: map[string]ValueState{
				"conditional_value": {
					Name:         "conditional_value",
					Value:        "default",
					DefaultValue: "default",
					IsOverridden: true,
					Proc: func(ctx map[string]any) any {
						// Simulates: return "premium" when user.tier == "premium"
						if tier, ok := ctx["user.tier"]; ok && tier == "premium" {
							return "premium_value"
						}
						return "default"
					},
				},
			},
		},
	}

	t.Run("value override for matching context", func(t *testing.T) {
		ctx := map[string]any{"user.tier": "premium"}
		val := flags.GetValue("conditional_value", ctx)
		if val != "premium_value" {
			t.Errorf("Expected 'premium_value', got %v", val)
		}
	})

	t.Run("default value for non-matching context", func(t *testing.T) {
		ctx := map[string]any{"user.tier": "free"}
		val := flags.GetValue("conditional_value", ctx)
		if val != "default" {
			t.Errorf("Expected 'default', got %v", val)
		}
	})
}

// Test GetValueInt and GetValueString
func TestGetValueIntAndString(t *testing.T) {
	logger := &testLogger{}
	flags := &FeatureFlags{
		logger: logger,
		state: State{
			valueState: map[string]ValueState{
				"int_value":    {Name: "int_value", Value: 42, DefaultValue: 10, IsOverridden: true},
				"float_value":  {Name: "float_value", Value: 3.14, DefaultValue: 1.0, IsOverridden: true},
				"string_value": {Name: "string_value", Value: "hello", DefaultValue: "default", IsOverridden: true},
				"wrong_type":   {Name: "wrong_type", Value: "not_an_int", DefaultValue: 5, IsOverridden: true},
			},
		},
	}

	t.Run("GetValueInt - success with int", func(t *testing.T) {
		val, err := flags.GetValueInt("int_value", nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})

	t.Run("GetValueInt - success with float64", func(t *testing.T) {
		val, err := flags.GetValueInt("float_value", nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != 3 {
			t.Errorf("Expected 3, got %d", val)
		}
	})

	t.Run("GetValueInt - error on wrong type", func(t *testing.T) {
		_, err := flags.GetValueInt("string_value", nil)
		if err == nil {
			t.Error("Expected error for wrong type")
		}
	})

	t.Run("GetValueInt - error on non-existent", func(t *testing.T) {
		_, err := flags.GetValueInt("non_existent", nil)
		if err == nil {
			t.Error("Expected error for non-existent value")
		}
	})

	t.Run("GetValueString - success", func(t *testing.T) {
		val, err := flags.GetValueString("string_value", nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("Expected 'hello', got %s", val)
		}
	})

	t.Run("GetValueString - error on wrong type", func(t *testing.T) {
		_, err := flags.GetValueString("int_value", nil)
		if err == nil {
			t.Error("Expected error for wrong type")
		}
	})

	t.Run("GetValueString - error on non-existent", func(t *testing.T) {
		_, err := flags.GetValueString("non_existent", nil)
		if err == nil {
			t.Error("Expected error for non-existent value")
		}
	})
}

// Test MustGetValueInt and MustGetValueString
func TestMustGetValueIntAndString(t *testing.T) {
	logger := &testLogger{}
	flags := &FeatureFlags{
		logger: logger,
		state: State{
			valueState: map[string]ValueState{
				"int_value":      {Name: "int_value", Value: 42, DefaultValue: 10, IsOverridden: true},
				"float_value":    {Name: "float_value", Value: 3.14, DefaultValue: 1.0, IsOverridden: true},
				"string_value":   {Name: "string_value", Value: "hello", DefaultValue: "default", IsOverridden: true},
				"wrong_type_int": {Name: "wrong_type_int", Value: "not_an_int", DefaultValue: 99, IsOverridden: true},
				"wrong_type_str": {Name: "wrong_type_str", Value: 123, DefaultValue: "fallback", IsOverridden: true},
			},
		},
	}

	t.Run("MustGetValueInt - success with int", func(t *testing.T) {
		val := flags.MustGetValueInt("int_value", nil)
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})

	t.Run("MustGetValueInt - success with float64", func(t *testing.T) {
		val := flags.MustGetValueInt("float_value", nil)
		if val != 3 {
			t.Errorf("Expected 3, got %d", val)
		}
	})

	t.Run("MustGetValueInt - fallback to default on wrong type", func(t *testing.T) {
		val := flags.MustGetValueInt("wrong_type_int", nil)
		if val != 99 {
			t.Errorf("Expected default 99, got %d", val)
		}
	})

	t.Run("MustGetValueInt - panic on non-existent", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for non-existent value")
			}
		}()
		flags.MustGetValueInt("non_existent", nil)
	})

	t.Run("MustGetValueString - success", func(t *testing.T) {
		val := flags.MustGetValueString("string_value", nil)
		if val != "hello" {
			t.Errorf("Expected 'hello', got %s", val)
		}
	})

	t.Run("MustGetValueString - fallback to default on wrong type", func(t *testing.T) {
		val := flags.MustGetValueString("wrong_type_str", nil)
		if val != "fallback" {
			t.Errorf("Expected default 'fallback', got %s", val)
		}
	})

	t.Run("MustGetValueString - panic on non-existent", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for non-existent value")
			}
		}()
		flags.MustGetValueString("non_existent", nil)
	})
}

// Test IsValueOverridden
func TestIsValueOverridden(t *testing.T) {
	flags := &FeatureFlags{
		state: State{
			valueState: map[string]ValueState{
				"overridden": {Name: "overridden", Value: 100, DefaultValue: 50, IsOverridden: true},
				"default":    {Name: "default", Value: 50, DefaultValue: 50, IsOverridden: false},
			},
		},
	}

	t.Run("value is overridden", func(t *testing.T) {
		if !flags.IsValueOverridden("overridden") {
			t.Error("Expected overridden to be true")
		}
	})

	t.Run("value is not overridden", func(t *testing.T) {
		if flags.IsValueOverridden("default") {
			t.Error("Expected default to be false")
		}
	})

	t.Run("non-existent value", func(t *testing.T) {
		if flags.IsValueOverridden("non_existent") {
			t.Error("Expected non_existent to be false")
		}
	})
}
