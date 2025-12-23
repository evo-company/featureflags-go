package featureflags

import "fmt"

// ValueState stores the state of a feature value including its evaluation function
type ValueState struct {
	Name         string
	Value        any       // current value (from server or default)
	DefaultValue any       // original default value
	IsOverridden bool      // true if value was set by server
	Proc         ValueProc // compiled condition evaluator (nil if using default)
}

// getValueState evaluates a value against a context and returns its value
func (state *State) getValueState(name string, ctx map[string]any) any {
	valueState, found := state.valueState[name]
	if !found {
		return nil
	}

	// If we have a proc (conditions from server), use it
	if valueState.Proc != nil {
		return valueState.Proc(ctx)
	}

	// Otherwise return the default/static value
	return valueState.Value
}

// GetValue returns the value evaluated against the provided context
func (flags *FeatureFlags) GetValue(name string, ctx map[string]any) any {
	flags.mu.RLock()
	defer flags.mu.RUnlock()
	return flags.state.getValueState(name, ctx)
}

// GetValueInt returns the value as an int. Returns an error if the value doesn't exist
// or cannot be cast to int.
func (flags *FeatureFlags) GetValueInt(name string, ctx map[string]any) (int, error) {
	flags.mu.RLock()
	defer flags.mu.RUnlock()

	value := flags.state.getValueState(name, ctx)
	if value == nil {
		return 0, fmt.Errorf("value %s not found", name)
	}

	// Try to cast to int
	if intVal, ok := value.(int); ok {
		return intVal, nil
	}

	// Try to cast to float64 (JSON numbers are decoded as float64)
	if floatVal, ok := value.(float64); ok {
		return int(floatVal), nil
	}

	return 0, fmt.Errorf("value %s cannot be cast to int (type: %T)", name, value)
}

// MustGetValueInt returns the value as an int. If the value cannot be cast to int,
// it returns the default value. Panics if the value key doesn't exist in the map
// (which indicates a programming error - asking for a value that was never defined).
func (flags *FeatureFlags) MustGetValueInt(name string, ctx map[string]any) int {
	flags.mu.RLock()
	defer flags.mu.RUnlock()

	valueState, exists := flags.state.valueState[name]
	if !exists {
		panic(fmt.Sprintf("value %s was never defined in defaults - this is a programming error", name))
	}

	value := flags.state.getValueState(name, ctx)

	// Try to cast current value to int
	if intVal, ok := value.(int); ok {
		return intVal
	}

	// Try to cast to float64 (JSON numbers are decoded as float64)
	if floatVal, ok := value.(float64); ok {
		return int(floatVal)
	}

	// Fall back to default value
	if defaultInt, ok := valueState.DefaultValue.(int); ok {
		flags.logger.Printf("Value %s cannot be cast to int, using default %d", name, defaultInt)
		return defaultInt
	}

	// This should never happen if defaults were properly initialized
	panic(fmt.Sprintf("value %s has no valid int default - this is a programming error", name))
}

// GetValueString returns the value as a string. Returns an error if the value doesn't exist
// or cannot be cast to string.
func (flags *FeatureFlags) GetValueString(name string, ctx map[string]any) (string, error) {
	flags.mu.RLock()
	defer flags.mu.RUnlock()

	value := flags.state.getValueState(name, ctx)
	if value == nil {
		return "", fmt.Errorf("value %s not found", name)
	}

	// Try to cast to string
	if strVal, ok := value.(string); ok {
		return strVal, nil
	}

	return "", fmt.Errorf("value %s cannot be cast to string (type: %T)", name, value)
}

// MustGetValueString returns the value as a string. If the value cannot be cast to string,
// it returns the default value. Panics if the value key doesn't exist in the map
// (which indicates a programming error - asking for a value that was never defined).
func (flags *FeatureFlags) MustGetValueString(name string, ctx map[string]any) string {
	flags.mu.RLock()
	defer flags.mu.RUnlock()

	valueState, exists := flags.state.valueState[name]
	if !exists {
		panic(fmt.Sprintf("value %s was never defined in defaults - this is a programming error", name))
	}

	value := flags.state.getValueState(name, ctx)

	// Try to cast current value to string
	if strVal, ok := value.(string); ok {
		return strVal
	}

	// Fall back to default value
	if defaultStr, ok := valueState.DefaultValue.(string); ok {
		flags.logger.Printf("Value %s cannot be cast to string, using default %s", name, defaultStr)
		return defaultStr
	}

	// This should never happen if defaults were properly initialized
	panic(fmt.Sprintf("value %s has no valid string default - this is a programming error", name))
}

// IsValueOverridden returns true if the value was set by the server, false if it's using the default.
func (flags *FeatureFlags) IsValueOverridden(name string) bool {
	flags.mu.RLock()
	defer flags.mu.RUnlock()

	if valueState, exists := flags.state.valueState[name]; exists {
		return valueState.IsOverridden
	}
	return false
}

// ValueResponse represents a value response from the server
type ValueResponse struct {
	Name          string           `json:"name"`
	Enabled       bool             `json:"enabled"`
	Overridden    bool             `json:"overridden"`
	ValueDefault  any              `json:"value_default"`
	ValueOverride any              `json:"value_override"`
	Conditions    []ValueCondition `json:"conditions"`
}

// ValueInput represents a value input for the load request
type ValueInput struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// Value represents a value definition with default
type Value struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}
