package featureflags

import (
	"testing"
)

// Test hashFlagValue function
func TestHashFlagValue(t *testing.T) {
	// Test deterministic behavior
	hash1 := hashFlagValue("user.id", 123)
	hash2 := hashFlagValue("user.id", 123)
	if hash1 != hash2 {
		t.Error("Expected hash to be deterministic")
	}

	// Test different values produce different hashes
	hash3 := hashFlagValue("user.id", 456)
	if hash1 == hash3 {
		t.Error("Expected different values to produce different hashes")
	}

	// Test different names produce different hashes
	hash4 := hashFlagValue("user.email", 123)
	if hash1 == hash4 {
		t.Error("Expected different names to produce different hashes")
	}
}

// Test opEqual operator
func TestOpEqual(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"equal strings", "user.name", "alice", map[string]any{"user.name": "alice"}, true},
		{"unequal strings", "user.name", "alice", map[string]any{"user.name": "bob"}, false},
		{"equal ints", "user.id", 123, map[string]any{"user.id": 123}, true},
		{"unequal ints", "user.id", 123, map[string]any{"user.id": 456}, false},
		{"equal floats", "user.id", 123.45, map[string]any{"user.id": 123.45}, true},
		{"unequal floats", "user.id", 123.45, map[string]any{"user.id": 456.78}, false},
		{"missing variable", "user.id", 123, map[string]any{}, false},
		{"nil context", "user.id", 123, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opEqual(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opLessThan operator
func TestOpLessThan(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"less than true (int)", "age", 30, map[string]any{"age": 25}, true},
		{"less than false (int)", "age", 30, map[string]any{"age": 35}, false},
		{"equal false (int)", "age", 30, map[string]any{"age": 30}, false},
		{"less than true (float)", "score", 10.5, map[string]any{"score": 5.5}, true},
		{"less than false (float)", "score", 10.5, map[string]any{"score": 15.5}, false},
		{"equal false (float)", "score", 10.5, map[string]any{"score": 10.5}, false},
		{"missing variable", "age", 30, map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opLessThan(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opLessOrEqual operator
func TestOpLessOrEqual(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"less than true", "age", 30, map[string]any{"age": 25}, true},
		{"equal true", "age", 30, map[string]any{"age": 30}, true},
		{"greater than false", "age", 30, map[string]any{"age": 35}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opLessOrEqual(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opGreaterThan operator
func TestOpGreaterThan(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"greater than true", "age", 30, map[string]any{"age": 35}, true},
		{"greater than false", "age", 30, map[string]any{"age": 25}, false},
		{"equal false", "age", 30, map[string]any{"age": 30}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opGreaterThan(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opGreaterOrEqual operator
func TestOpGreaterOrEqual(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"greater than true", "age", 30, map[string]any{"age": 35}, true},
		{"equal true", "age", 30, map[string]any{"age": 30}, true},
		{"less than false", "age", 30, map[string]any{"age": 25}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opGreaterOrEqual(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opContains operator
func TestOpContains(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"contains substring", "email", "@example.com", map[string]any{"email": "user@example.com"}, true},
		{"does not contain", "email", "@other.com", map[string]any{"email": "user@example.com"}, false},
		{"missing variable", "email", "@example.com", map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opContains(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opPercent operator
func TestOpPercent(t *testing.T) {
	// Test that percent works correctly for rollout
	fn := opPercent("user.id", 50)

	// Count how many of 1000 users pass the check
	passCount := 0
	for i := 0; i < 1000; i++ {
		ctx := map[string]any{"user.id": i}
		if fn(ctx) {
			passCount++
		}
	}

	// Should be roughly 50% (allow some variance due to hash distribution)
	if passCount < 400 || passCount > 600 {
		t.Errorf("Expected ~50%% pass rate, got %d/1000", passCount)
	}

	// Test 0% rollout
	fn0 := opPercent("user.id", 0)
	for i := 0; i < 100; i++ {
		ctx := map[string]any{"user.id": i}
		if fn0(ctx) {
			t.Error("Expected 0% rollout to always return false")
			break
		}
	}

	// Test 100% rollout
	fn100 := opPercent("user.id", 100)
	for i := 0; i < 100; i++ {
		ctx := map[string]any{"user.id": i}
		if !fn100(ctx) {
			t.Error("Expected 100% rollout to always return true")
			break
		}
	}
}

// Test opRegexp operator
func TestOpRegexp(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"matches regex", "email", `^[a-z]+@example\.com$`, map[string]any{"email": "user@example.com"}, true},
		{"does not match", "email", `^[a-z]+@example\.com$`, map[string]any{"email": "user@other.com"}, false},
		{"partial match", "email", `example`, map[string]any{"email": "user@example.com"}, true},
		{"missing variable", "email", `^test`, map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opRegexp(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}

	// Test invalid regex returns false
	t.Run("invalid regex", func(t *testing.T) {
		fn := opRegexp("email", `[invalid`)
		result := fn(map[string]any{"email": "test"})
		if result {
			t.Error("Expected invalid regex to return false")
		}
	})
}

// Test opWildcard operator
func TestOpWildcard(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"matches wildcard", "email", "*@example.com", map[string]any{"email": "user@example.com"}, true},
		{"matches prefix wildcard", "path", "/api/*", map[string]any{"path": "/api/users"}, true},
		{"matches suffix wildcard", "file", "*.txt", map[string]any{"file": "document.txt"}, true},
		{"matches middle wildcard", "email", "user*@example.com", map[string]any{"email": "user123@example.com"}, true},
		{"does not match", "email", "*@example.com", map[string]any{"email": "user@other.com"}, false},
		{"exact match", "name", "alice", map[string]any{"name": "alice"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opWildcard(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opSubset operator
func TestOpSubset(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"subset true", "roles", []any{"admin", "user", "guest"}, map[string]any{"roles": []string{"admin", "user"}}, true},
		{"not a subset", "roles", []any{"admin"}, map[string]any{"roles": []string{"admin", "user"}}, false},
		{"exact match", "roles", []any{"admin", "user"}, map[string]any{"roles": []string{"admin", "user"}}, true},
		{"empty value", "roles", []any{}, map[string]any{"roles": []string{"admin"}}, false},
		{"empty context", "roles", []any{"admin"}, map[string]any{"roles": []string{}}, false},
		{"missing variable", "roles", []any{"admin"}, map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opSubset(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test opSuperset operator
func TestOpSuperset(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		value    any
		ctx      map[string]any
		expected bool
	}{
		{"superset true", "roles", []any{"admin"}, map[string]any{"roles": []string{"admin", "user", "guest"}}, true},
		{"not a superset", "roles", []any{"admin", "user"}, map[string]any{"roles": []string{"admin"}}, false},
		{"exact match", "roles", []any{"admin", "user"}, map[string]any{"roles": []string{"admin", "user"}}, true},
		{"empty value", "roles", []any{}, map[string]any{"roles": []string{"admin"}}, false},
		{"missing variable", "roles", []any{"admin"}, map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := opSuperset(tt.varName, tt.value)
			result := fn(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test checkProc function
func TestCheckProc(t *testing.T) {
	t.Run("nil value returns false", func(t *testing.T) {
		check := Check{
			Operator: OpEqual,
			Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
			Value:    nil,
		}
		fn := checkProc(check)
		if fn(map[string]any{"user.id": 123}) {
			t.Error("Expected nil value check to return false")
		}
	})

	t.Run("unknown operator returns false", func(t *testing.T) {
		check := Check{
			Operator: Operator(999), // Unknown operator
			Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
			Value:    123,
		}
		fn := checkProc(check)
		if fn(map[string]any{"user.id": 123}) {
			t.Error("Expected unknown operator to return false")
		}
	})

	t.Run("valid check", func(t *testing.T) {
		check := Check{
			Operator: OpEqual,
			Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
			Value:    float64(123),
		}
		fn := checkProc(check)
		if !fn(map[string]any{"user.id": float64(123)}) {
			t.Error("Expected valid check to return true")
		}
	})
}

// Test flagProc function
func TestFlagProc(t *testing.T) {
	t.Run("not overridden returns nil", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    true,
			Overridden: false,
		}
		proc := flagProc(flag)
		if proc != nil {
			t.Error("Expected non-overridden flag to return nil proc")
		}
	})

	t.Run("disabled flag without conditions", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    false,
			Overridden: true,
		}
		proc := flagProc(flag)
		if proc(map[string]any{}) {
			t.Error("Expected disabled flag to return false")
		}
	})

	t.Run("enabled flag without conditions", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    true,
			Overridden: true,
		}
		proc := flagProc(flag)
		if !proc(map[string]any{}) {
			t.Error("Expected enabled flag without conditions to return true")
		}
	})

	t.Run("flag with single condition", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    true,
			Overridden: true,
			Conditions: []Condition{
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
							Value:    float64(123),
						},
					},
				},
			},
		}
		proc := flagProc(flag)

		// Should be true for matching context
		if !proc(map[string]any{"user.id": float64(123)}) {
			t.Error("Expected flag to be true for matching context")
		}

		// Should be false for non-matching context
		if proc(map[string]any{"user.id": float64(456)}) {
			t.Error("Expected flag to be false for non-matching context")
		}
	})

	t.Run("flag with multiple conditions (OR logic)", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    true,
			Overridden: true,
			Conditions: []Condition{
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
							Value:    float64(123),
						},
					},
				},
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
							Value:    float64(456),
						},
					},
				},
			},
		}
		proc := flagProc(flag)

		// Should be true for first condition
		if !proc(map[string]any{"user.id": float64(123)}) {
			t.Error("Expected flag to be true for first condition")
		}

		// Should be true for second condition
		if !proc(map[string]any{"user.id": float64(456)}) {
			t.Error("Expected flag to be true for second condition")
		}

		// Should be false for neither condition
		if proc(map[string]any{"user.id": float64(789)}) {
			t.Error("Expected flag to be false for neither condition")
		}
	})

	t.Run("flag with multiple checks in condition (AND logic)", func(t *testing.T) {
		flag := FlagResponse{
			Name:       "test_flag",
			Enabled:    true,
			Overridden: true,
			Conditions: []Condition{
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.id", Type: TypeNumber},
							Value:    float64(123),
						},
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.tier", Type: TypeString},
							Value:    "premium",
						},
					},
				},
			},
		}
		proc := flagProc(flag)

		// Should be true when both checks pass
		if !proc(map[string]any{"user.id": float64(123), "user.tier": "premium"}) {
			t.Error("Expected flag to be true when all checks pass")
		}

		// Should be false when only one check passes
		if proc(map[string]any{"user.id": float64(123), "user.tier": "free"}) {
			t.Error("Expected flag to be false when not all checks pass")
		}
	})
}

// Test valueProc function
func TestValueProc(t *testing.T) {
	t.Run("not overridden returns default", func(t *testing.T) {
		value := ValueResponse{
			Name:         "test_value",
			Overridden:   false,
			ValueDefault: "default_value",
		}
		proc := valueProc(value)
		result := proc(map[string]any{})
		if result != "default_value" {
			t.Errorf("Expected default_value, got %v", result)
		}
	})

	t.Run("disabled value returns override", func(t *testing.T) {
		value := ValueResponse{
			Name:          "test_value",
			Enabled:       false,
			Overridden:    true,
			ValueOverride: "override_value",
		}
		proc := valueProc(value)
		result := proc(map[string]any{})
		if result != "override_value" {
			t.Errorf("Expected override_value, got %v", result)
		}
	})

	t.Run("value with conditions", func(t *testing.T) {
		value := ValueResponse{
			Name:          "test_value",
			Enabled:       true,
			Overridden:    true,
			ValueOverride: "base_override",
			Conditions: []ValueCondition{
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.tier", Type: TypeString},
							Value:    "premium",
						},
					},
					ValueOverride: "premium_value",
				},
				{
					Checks: []Check{
						{
							Operator: OpEqual,
							Variable: CheckVariable{Name: "user.tier", Type: TypeString},
							Value:    "enterprise",
						},
					},
					ValueOverride: "enterprise_value",
				},
			},
		}
		proc := valueProc(value)

		// Should return first matching condition's value
		result := proc(map[string]any{"user.tier": "premium"})
		if result != "premium_value" {
			t.Errorf("Expected premium_value, got %v", result)
		}

		// Should return second condition's value
		result = proc(map[string]any{"user.tier": "enterprise"})
		if result != "enterprise_value" {
			t.Errorf("Expected enterprise_value, got %v", result)
		}

		// Should return base override for non-matching
		result = proc(map[string]any{"user.tier": "free"})
		if result != "base_override" {
			t.Errorf("Expected base_override, got %v", result)
		}
	})
}
