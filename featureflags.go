package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultSyncInterval = 10 * time.Second
)

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

type ValueState struct {
	Name         string
	Value        interface{} // current value (from server or default)
	DefaultValue interface{} // original default value
	IsOverridden bool        // true if value was set by server
}

type State struct {
	flagState  map[string]FlagState
	flagNames  []string
	valueState map[string]ValueState
	valueNames []string
	version    int
}

func (state *State) FlagState(name string) bool {
	result := false
	value, foundValue := state.flagState[name]

	if foundValue {
		result = value.Enabled
	}
	return result
}

func (state *State) ValueState(name string) interface{} {
	value, foundValue := state.valueState[name]

	if foundValue {
		return value.Value
	}
	return nil
}

func (state *State) Update(version int, flags []FlagResponse, values []ValueResponse) {
	if state.version == version {
		return
	}

	state.version = version
	for _, flag := range flags {
		state.flagState[flag.Name] = FlagState{
			Name:    flag.Name,
			Enabled: flag.Enabled,
		}
	}

	for _, value := range values {
		// Preserve the default value if it exists
		existingState, exists := state.valueState[value.Name]
		defaultVal := interface{}(nil)
		if exists {
			defaultVal = existingState.DefaultValue
		}

		state.valueState[value.Name] = ValueState{
			Name:         value.Name,
			Value:        value.Value,
			DefaultValue: defaultVal,
			IsOverridden: true, // Value came from server
		}
	}
}

type Logger interface {
	Fatalf(format string, args ...any)
	Printf(format string, args ...any)
}

// defaultLogger is a no-op logger used when no logger is provided
type defaultLogger struct{}

func (l *defaultLogger) Fatalf(format string, args ...any) {}
func (l *defaultLogger) Printf(format string, args ...any) {}

type FeatureFlags struct {
	client       *http.Client
	logger       Logger
	project      string
	state        State
	variables    []Variable
	httpAddr     string
	syncInterval time.Duration
}

func (flags *FeatureFlags) Get(name string) bool {
	return flags.state.FlagState(name)
}

func (flags *FeatureFlags) GetValue(name string) interface{} {
	return flags.state.ValueState(name)
}

// GetValueInt returns the value as an int. Returns an error if the value doesn't exist
// or cannot be cast to int.
func (flags *FeatureFlags) GetValueInt(name string) (int, error) {
	value := flags.state.ValueState(name)
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
func (flags *FeatureFlags) MustGetValueInt(name string) int {
	valueState, exists := flags.state.valueState[name]
	if !exists {
		panic(fmt.Sprintf("value %s was never defined in defaults - this is a programming error", name))
	}

	value := valueState.Value

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
func (flags *FeatureFlags) GetValueString(name string) (string, error) {
	value := flags.state.ValueState(name)
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
func (flags *FeatureFlags) MustGetValueString(name string) string {
	valueState, exists := flags.state.valueState[name]
	if !exists {
		panic(fmt.Sprintf("value %s was never defined in defaults - this is a programming error", name))
	}

	value := valueState.Value

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
	if valueState, exists := flags.state.valueState[name]; exists {
		return valueState.IsOverridden
	}
	return false
}

func (flags *FeatureFlags) SyncLoop() {
	for {
		time.Sleep(flags.syncInterval)
		err := flags.Sync()
		if err != nil {
			flags.logger.Printf("Could not sync flags: %v", err)
		} else {
			flags.logger.Printf("Flags has been synced")
		}
	}
}

var ErrorCantSyncFlags = errors.New("can not sync flags")

func (flags *FeatureFlags) Sync() error {
	res, err := flags.SyncRequest()
	if err != nil {
		return errors.Join(ErrorCantSyncFlags, err)
	}

	flags.state.Update(res.Version, res.Flags, res.Values)
	return nil
}

// TODO split into files

type SyncFlagsRequest struct {
	Project string   `json:"project"`
	Version int      `json:"version"`
	Flags   []string `json:"flags"`
	Values  []string `json:"values"`
}

type FlagResponse struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type ValueResponse struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // Using interface{} for Any type
}

type SyncFlagsResponse struct {
	Version int             `json:"version"`
	Flags   []FlagResponse  `json:"flags"`
	Values  []ValueResponse `json:"values"`
}

type ValueInput struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type LoadFlagsRequest struct {
	Project   string       `json:"project"`
	Version   int          `json:"version"`
	Variables []Variable   `json:"variables"`
	Flags     []string     `json:"flags"`
	Values    []ValueInput `json:"values"`
}

type LoadFlagsResponse struct {
	Version int             `json:"version"`
	Flags   []FlagResponse  `json:"flags"`
	Values  []ValueResponse `json:"values"`
}

func (flags *FeatureFlags) SyncRequest() (*SyncFlagsResponse, error) {
	req := SyncFlagsRequest{
		Project: flags.project,
		Version: flags.state.version,
		Flags:   flags.state.flagNames,
		Values:  flags.state.valueNames,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/flags/sync", flags.httpAddr)
	res, err := flags.client.Post(
		url, "application/json", bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http request to %s failed with status: %s", url, res.Status)
	}

	var reply SyncFlagsResponse
	err = json.NewDecoder(res.Body).Decode(&reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

// LoadRequest sends a load request to the feature flags server.
// This creates a project on the server if it doesn't exist, initializes flags, values, and variables,
// and syncs the current project state from server to client.
func (flags *FeatureFlags) LoadRequest() (*LoadFlagsResponse, error) {
	// Build value inputs from current state
	valueInputs := make([]ValueInput, 0, len(flags.state.valueState))
	for _, valueState := range flags.state.valueState {
		valueInputs = append(valueInputs, ValueInput{
			Name:  valueState.Name,
			Value: valueState.Value,
		})
	}

	req := LoadFlagsRequest{
		Project:   flags.project,
		Version:   flags.state.version,
		Variables: flags.variables,
		Flags:     flags.state.flagNames,
		Values:    valueInputs,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/flags/load", flags.httpAddr)
	res, err := flags.client.Post(
		url, "application/json", bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http request to %s failed with status: %s", url, res.Status)
	}

	var reply LoadFlagsResponse
	err = json.NewDecoder(res.Body).Decode(&reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

var ErrorCantLoadFlags = errors.New("can not load flags")

// Load initializes the project on the server by creating it if it doesn't exist,
// creating and initializing flags, values, and variables, and syncing the current
// project state from the server to the client.
func (flags *FeatureFlags) Load() error {
	res, err := flags.LoadRequest()
	if err != nil {
		return errors.Join(ErrorCantLoadFlags, err)
	}

	flags.state.Update(res.Version, res.Flags, res.Values)
	return nil
}

type VariableType int

const (
	TypeString    VariableType = iota + 1 // 1
	TypeNumber                            // 2
	TypeTimestamp                         // 3
	TypeSet                               // 4
)

type Variable struct {
	Name string       `json:"name"`
	Type VariableType `json:"type"`
}

type Flag struct { // TODO: do we need json tags here ?
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Value struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // Using interface{} for Any type
}

type Defaults struct {
	Flags  []Flag
	Values []Value
}

// ClientConfig holds configuration options for the FeatureFlags client
type ClientConfig struct {
	variables    []Variable
	syncInterval time.Duration
	logger       Logger
}

// ClientOption is a function that configures a ClientConfig
type ClientOption func(*ClientConfig)

// WithVariables sets the variables for targeting rules
func WithVariables(variables []Variable) ClientOption {
	return func(c *ClientConfig) {
		c.variables = variables
	}
}

// WithSyncInterval sets the interval for syncing flags
func WithSyncInterval(interval time.Duration) ClientOption {
	return func(c *ClientConfig) {
		c.syncInterval = interval
	}
}

// WithLogger sets the logger for the client
func WithLogger(logger Logger) ClientOption {
	return func(c *ClientConfig) {
		c.logger = logger
	}
}

func MakeClient(
	ctx context.Context,
	httpAddr string,
	project string,
	defaults Defaults,
	opts ...ClientOption,
) (*FeatureFlags, error) {
	// Initialize config with defaults
	config := &ClientConfig{
		syncInterval: defaultSyncInterval,
		logger:       nil, // Will use a default logger if nil
		variables:    make([]Variable, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Use default logger if none provided
	if config.logger == nil {
		config.logger = &defaultLogger{}
	}

	client := &http.Client{}
	flagsMap := make(map[string]FlagState, len(defaults.Flags))
	flagNames := make([]string, len(defaults.Flags))
	valuesMap := make(map[string]ValueState, len(defaults.Values))
	valueNames := make([]string, len(defaults.Values))

	for i, flag := range defaults.Flags {
		flagsMap[flag.Name] = FlagState{
			Name:    flag.Name,
			Enabled: flag.Enabled,
		}
		flagNames[i] = flag.Name
	}

	for i, value := range defaults.Values {
		valuesMap[value.Name] = ValueState{
			Name:         value.Name,
			Value:        value.Value,
			DefaultValue: value.Value,
			IsOverridden: false,
		}
		valueNames[i] = value.Name
	}

	if config.syncInterval <= 0 {
		config.syncInterval = defaultSyncInterval
	}

	flagsClient := FeatureFlags{
		client:    client,
		project:   project,
		httpAddr:  httpAddr,
		variables: config.variables,
		state: State{
			flagState:  flagsMap,
			flagNames:  flagNames,
			valueState: valuesMap,
			valueNames: valueNames,
		},
		logger:       config.logger,
		syncInterval: config.syncInterval,
	}
	// Load will create a project on the server if it doesn't exist,
	// create and initialize flags, values and variables, and will sync
	// current project state from server to client
	err := flagsClient.Load()
	if err != nil {
		return nil, err
	}
	go flagsClient.SyncLoop()
	return &flagsClient, nil
}
