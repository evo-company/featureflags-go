package featureflags

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
)

// Operator represents the type of comparison operation for a condition check
type Operator int

const (
	OpEqual          Operator = 1
	OpLessThan       Operator = 2
	OpLessOrEqual    Operator = 3
	OpGreaterThan    Operator = 4
	OpGreaterOrEqual Operator = 5
	OpContains       Operator = 6
	OpPercent        Operator = 7
	OpRegexp         Operator = 8
	OpWildcard       Operator = 9
	OpSubset         Operator = 10
	OpSuperset       Operator = 11
)

// CheckVariable represents a variable used in condition checks
type CheckVariable struct {
	Name string       `json:"name"`
	Type VariableType `json:"type"`
}

// Check represents a single condition check
type Check struct {
	Operator Operator      `json:"operator"`
	Variable CheckVariable `json:"variable"`
	Value    any           `json:"value"`
}

// Condition represents a condition with multiple checks (AND logic)
type Condition struct {
	Checks []Check `json:"checks"`
}

// ValueCondition represents a condition for values with an override value
type ValueCondition struct {
	Checks        []Check `json:"checks"`
	ValueOverride any     `json:"value_override"`
}

// hashFlagValue generates a deterministic hash for percent-based conditions
func hashFlagValue(name string, value any) uint32 {
	data := fmt.Sprintf("%s%v", name, value)
	hash := md5.Sum([]byte(data))
	// Take last 4 bytes as little-endian uint32 (matching Python implementation)
	return binary.LittleEndian.Uint32(hash[12:16])
}

// toFloat64 attempts to convert a value to float64 for numeric comparisons
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case string:
		return 0, false
	default:
		return 0, false
	}
}

// toString attempts to convert a value to string
func toString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// toStringSlice attempts to convert a value to []string for set operations
func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			s, ok := toString(item)
			if !ok {
				return nil, false
			}
			result[i] = s
		}
		return result, true
	default:
		return nil, false
	}
}

// toSet converts a slice to a map for set operations
func toSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, item := range slice {
		set[item] = struct{}{}
	}
	return set
}

// CheckFunc is a function that evaluates a check against a context
type CheckFunc func(ctx map[string]any) bool

// FlagProc is a function that evaluates a flag against a context
type FlagProc func(ctx map[string]any) bool

// ValueProc is a function that evaluates a value against a context and returns the result
type ValueProc func(ctx map[string]any) any

// opEqual creates a check function for equality comparison
func opEqual(name string, value any) CheckFunc {
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		return ctxVal == value
	}
}

// opLessThan creates a check function for less-than comparison
func opLessThan(name string, value any) CheckFunc {
	targetVal, targetOk := toFloat64(value)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxFloat, ctxOk := toFloat64(ctxVal)
		if !targetOk || !ctxOk {
			// Fall back to string comparison
			ctxStr, _ := toString(ctxVal)
			valStr, _ := toString(value)
			return ctxStr < valStr
		}
		return ctxFloat < targetVal
	}
}

// opLessOrEqual creates a check function for less-than-or-equal comparison
func opLessOrEqual(name string, value any) CheckFunc {
	targetVal, targetOk := toFloat64(value)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxFloat, ctxOk := toFloat64(ctxVal)
		if !targetOk || !ctxOk {
			ctxStr, _ := toString(ctxVal)
			valStr, _ := toString(value)
			return ctxStr <= valStr
		}
		return ctxFloat <= targetVal
	}
}

// opGreaterThan creates a check function for greater-than comparison
func opGreaterThan(name string, value any) CheckFunc {
	targetVal, targetOk := toFloat64(value)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxFloat, ctxOk := toFloat64(ctxVal)
		if !targetOk || !ctxOk {
			ctxStr, _ := toString(ctxVal)
			valStr, _ := toString(value)
			return ctxStr > valStr
		}
		return ctxFloat > targetVal
	}
}

// opGreaterOrEqual creates a check function for greater-than-or-equal comparison
func opGreaterOrEqual(name string, value any) CheckFunc {
	targetVal, targetOk := toFloat64(value)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxFloat, ctxOk := toFloat64(ctxVal)
		if !targetOk || !ctxOk {
			ctxStr, _ := toString(ctxVal)
			valStr, _ := toString(value)
			return ctxStr >= valStr
		}
		return ctxFloat >= targetVal
	}
}

// opContains creates a check function for substring containment
func opContains(name string, value any) CheckFunc {
	valStr, _ := toString(value)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxStr, _ := toString(ctxVal)
		return strings.Contains(ctxStr, valStr)
	}
}

// opPercent creates a check function for percentage-based rollout
func opPercent(name string, value any) CheckFunc {
	threshold, ok := toFloat64(value)
	if !ok {
		return func(ctx map[string]any) bool { return false }
	}
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		hash := hashFlagValue(name, ctxVal)
		return hash%100 < uint32(threshold)
	}
}

// opRegexp creates a check function for regular expression matching
func opRegexp(name string, value any) CheckFunc {
	pattern, _ := toString(value)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return func(ctx map[string]any) bool { return false }
	}
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxStr, _ := toString(ctxVal)
		return re.MatchString(ctxStr)
	}
}

// opWildcard creates a check function for wildcard pattern matching
// Wildcards use * to match any sequence of characters
func opWildcard(name string, value any) CheckFunc {
	pattern, _ := toString(value)
	// Convert wildcard pattern to regex: escape special chars, replace * with .*
	parts := strings.Split(pattern, "*")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	regexPattern := "^" + strings.Join(parts, "(?:.*)") + "$"
	return opRegexp(name, regexPattern)
}

// opSubset (a.k.a Included in) creates a check function for subset comparison
// Returns true if value from ctx[name] is subset of value from server
// (in other words, ctx[name] included in value_set from server)
func opSubset(name string, value any) CheckFunc {
	valSlice, ok := toStringSlice(value)
	if !ok || len(valSlice) == 0 {
		return func(ctx map[string]any) bool { return false }
	}
	valSet := toSet(valSlice)
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxSlice, ok := toStringSlice(ctxVal)
		if !ok || len(ctxSlice) == 0 {
			return false
		}
		// Check if valSet is a superset of ctxSlice (all ctx items in val)
		for _, item := range ctxSlice {
			if _, exists := valSet[item]; !exists {
				return false
			}
		}
		return true
	}
}

// opSuperset (a.k.a Includes) creates a check function for superset comparison
// Returns true if value from ctx[name] is superset of value from server
// (in other words, ctx[name] includes all value items from server)
func opSuperset(name string, serverValue any) CheckFunc {
	serverValueSlice, ok := toStringSlice(serverValue)
	if !ok || len(serverValueSlice) == 0 {
		return func(ctx map[string]any) bool { return false }
	}
	return func(ctx map[string]any) bool {
		ctxVal, ok := ctx[name]
		if !ok {
			return false
		}
		ctxSlice, ok := toStringSlice(ctxVal)
		if !ok || len(ctxSlice) == 0 {
			return false
		}
		ctxSet := toSet(ctxSlice)
		// Check if serverValueSlice is a subset of ctxSet (all val items in ctx)
		for _, item := range serverValueSlice {
			if _, exists := ctxSet[item]; !exists {
				return false
			}
		}
		return true
	}
}

// operatorFuncs maps operators to their implementation functions
var operatorFuncs = map[Operator]func(name string, value any) CheckFunc{
	OpEqual:          opEqual,
	OpLessThan:       opLessThan,
	OpLessOrEqual:    opLessOrEqual,
	OpGreaterThan:    opGreaterThan,
	OpGreaterOrEqual: opGreaterOrEqual,
	OpContains:       opContains,
	OpPercent:        opPercent,
	OpRegexp:         opRegexp,
	OpWildcard:       opWildcard,
	OpSubset:         opSubset,
	OpSuperset:       opSuperset,
}

// checkProc creates a check function from a Check definition
func checkProc(check Check) CheckFunc {
	if check.Value == nil {
		return func(ctx map[string]any) bool { return false }
	}
	opFunc, ok := operatorFuncs[check.Operator]
	if !ok {
		return func(ctx map[string]any) bool { return false }
	}
	return opFunc(check.Variable.Name, check.Value)
}

// flagProc creates a flag evaluation function from a FlagResponse
// Returns nil if the flag was not overridden (use default value)
func flagProc(flag FlagResponse) FlagProc {
	if !flag.Overridden {
		// Flag was not overridden on server, use default value
		return nil
	}

	// Build condition check functions
	var conditions [][]CheckFunc
	for _, condition := range flag.Conditions {
		var checks []CheckFunc
		for _, check := range condition.Checks {
			checks = append(checks, checkProc(check))
		}
		// Invalid condition (empty checks) becomes a false check
		if len(checks) == 0 {
			checks = []CheckFunc{func(ctx map[string]any) bool { return false }}
		}
		conditions = append(conditions, checks)
	}

	// If flag is enabled and has conditions, evaluate them
	if flag.Enabled && len(conditions) > 0 {
		return func(ctx map[string]any) bool {
			// OR of ANDs: any condition where all checks pass
			for _, checks := range conditions {
				allPass := true
				for _, check := range checks {
					if !check(ctx) {
						allPass = false
						break
					}
				}
				if allPass {
					return true
				}
			}
			return false
		}
	}

	// Flag is disabled or has no conditions, return static value
	return func(ctx map[string]any) bool {
		return flag.Enabled
	}
}

// valueProc creates a value evaluation function from a ValueResponse
func valueProc(value ValueResponse) ValueProc {
	if !value.Overridden {
		// Value was not overridden on server, use default value
		return func(ctx map[string]any) any {
			return value.ValueDefault
		}
	}

	// Build condition check functions with their override values
	type conditionWithValue struct {
		checks        []CheckFunc
		valueOverride any
	}
	var conditions []conditionWithValue

	for _, condition := range value.Conditions {
		var checks []CheckFunc
		for _, check := range condition.Checks {
			checks = append(checks, checkProc(check))
		}
		// Invalid condition (empty checks) becomes a false check
		if len(checks) == 0 {
			checks = []CheckFunc{func(ctx map[string]any) bool { return false }}
		}
		conditions = append(conditions, conditionWithValue{
			checks:        checks,
			valueOverride: condition.ValueOverride,
		})
	}

	// If value is enabled and has conditions, evaluate them
	if value.Enabled && len(conditions) > 0 {
		return func(ctx map[string]any) any {
			// Check conditions in order, return first matching override
			for _, cond := range conditions {
				allPass := true
				// If all checks for a contition pass, use override from this condition
				for _, check := range cond.checks {
					if !check(ctx) {
						allPass = false
						break
					}
				}
				if allPass {
					return cond.valueOverride
				}
			}
			// No condition matched, return the base override value
			return value.ValueOverride
		}
	}

	// Value is disabled or has no conditions, return override value
	return func(ctx map[string]any) any {
		return value.ValueOverride
	}
}
