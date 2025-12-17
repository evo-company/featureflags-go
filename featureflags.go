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
	Name  string
	Value interface{}
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
	// TODO: teach in docs that this must be handled as if not nil
	// TODO: or return some deafault ?
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
		state.valueState[value.Name] = ValueState{
			Name:  value.Name,
			Value: value.Value, // decoded on use
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
	Project   string        `json:"project"`
	Version   int           `json:"version"`
	Variables []Variable    `json:"variables"`
	Flags     []string      `json:"flags"`
	Values    []ValueInput  `json:"values"`
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
	TypeString VariableType = iota + 1
	TypeNumber
	TypeTimestamp
	TypeSet
)

type Variable struct {
	Name string
	Type VariableType
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
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Use default logger if none provided
	if config.logger == nil {
		config.logger = &defaultLogger{}
	}

	c := &http.Client{}
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
			Name:  value.Name,
			Value: nil, // This is a default value
			// TODO: maybe default must be a separate field
		}
		valueNames[i] = value.Name
	}

	if config.syncInterval <= 0 {
		config.syncInterval = defaultSyncInterval
	}

	flagsClient := FeatureFlags{
		client:    c,
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
