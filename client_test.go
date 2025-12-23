package featureflags

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test helpers

type testLogger struct {
	messages []string
}

func (l *testLogger) Printf(format string, args ...any) {
	// Store messages for verification in tests if needed
	l.messages = append(l.messages, format)
}

func (l *testLogger) Fatalf(format string, args ...any) {
	// Store messages for verification in tests if needed
	l.messages = append(l.messages, format)
}

// Test Load API call with mock HTTP server
func TestLoadRequest(t *testing.T) {
	t.Run("successful load request", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.URL.Path != "/flags/load" {
				t.Errorf("Expected path /flags/load, got %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			// Parse request body
			var req LoadFlagsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			// Verify request contents
			if req.Project != "test-project" {
				t.Errorf("Expected project 'test-project', got %s", req.Project)
			}

			// Send response
			resp := LoadFlagsResponse{
				Version: 1,
				Flags: []FlagResponse{
					{Name: "test_flag", Enabled: true, Overridden: true},
				},
				Values: []ValueResponse{
					{Name: "test_value", Enabled: true, Overridden: true, ValueOverride: "hello"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create FeatureFlags client
		flags := &FeatureFlags{
			client:   server.Client(),
			httpAddr: server.URL,
			project:  "test-project",
			logger:   &testLogger{},
			state: State{
				version:    0,
				flagState:  make(map[string]FlagState),
				flagNames:  []string{"test_flag"},
				valueState: make(map[string]ValueState),
				valueNames: []string{"test_value"},
			},
		}

		// Call LoadRequest
		resp, err := flags.LoadRequest()
		if err != nil {
			t.Fatalf("LoadRequest failed: %v", err)
		}

		// Verify response
		if resp.Version != 1 {
			t.Errorf("Expected version 1, got %d", resp.Version)
		}
		// Verify flags
		if len(resp.Flags) != 1 {
			t.Errorf("Expected 1 flag, got %d", len(resp.Flags))
		}
		if resp.Flags[0].Name != "test_flag" {
			t.Errorf("Expected flag name 'test_flag', got %s", resp.Flags[0].Name)
		}
		if !resp.Flags[0].Enabled {
			t.Error("Expected flag to be enabled")
		}

		// Verify values
		if len(resp.Values) != 1 {
			t.Errorf("Expected 1 value, got %d", len(resp.Flags))
		}
		if resp.Values[0].Name != "test_value" {
			t.Errorf("Expected value name 'test_flag', got %s", resp.Flags[0].Name)
		}
		if !resp.Values[0].Enabled {
			t.Error("Expected value to be enabled")
		}
		if !resp.Values[0].Overridden {
			t.Error("Expected value to be overridden")
		}

		if resp.Values[0].ValueOverride != "hello" {
			t.Error("Expected value to have overridden value 'hello'")
		}
	})

	t.Run("load request with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		flags := &FeatureFlags{
			client:   server.Client(),
			httpAddr: server.URL,
			project:  "test-project",
			logger:   &testLogger{},
			state: State{
				flagState:  make(map[string]FlagState),
				valueState: make(map[string]ValueState),
			},
		}

		_, err := flags.LoadRequest()
		if err == nil {
			t.Error("Expected error for server error response")
		}
	})
}

// Test Load method
func TestLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LoadFlagsResponse{
			Version: 2,
			Flags: []FlagResponse{
				{Name: "feature_a", Enabled: true, Overridden: true},
				{Name: "feature_b", Enabled: false, Overridden: true},
			},
			Values: []ValueResponse{
				{Name: "timeout", Enabled: true, Overridden: true, ValueOverride: float64(30)},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	flags := &FeatureFlags{
		client:   server.Client(),
		httpAddr: server.URL,
		project:  "test-project",
		logger:   &testLogger{},
		state: State{
			version:    0,
			flagState:  make(map[string]FlagState),
			flagNames:  []string{"feature_a", "feature_b"},
			valueState: make(map[string]ValueState),
			valueNames: []string{"timeout"},
		},
	}

	err := flags.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify state was updated
	if flags.state.version != 2 {
		t.Errorf("Expected version 2, got %d", flags.state.version)
	}
	if !flags.state.getFlagState("feature_a", nil) {
		t.Error("Expected feature_a to be enabled")
	}
	if flags.state.getFlagState("feature_b", nil) {
		t.Error("Expected feature_b to be disabled")
	}
}

// Test Sync API call with mock HTTP server
func TestSyncRequest(t *testing.T) {
	t.Run("successful sync request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/flags/sync" {
				t.Errorf("Expected path /flags/sync, got %s", r.URL.Path)
			}

			resp := SyncFlagsResponse{
				Version: 3,
				Flags: []FlagResponse{
					{Name: "sync_flag", Enabled: true, Overridden: true},
				},
				Values: []ValueResponse{
					{Name: "sync_value", Enabled: true, Overridden: true, ValueOverride: 42.0},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		flags := &FeatureFlags{
			client:   server.Client(),
			httpAddr: server.URL,
			project:  "test-project",
			logger:   &testLogger{},
			state: State{
				version:    2,
				flagState:  make(map[string]FlagState),
				flagNames:  []string{"sync_flag"},
				valueState: make(map[string]ValueState),
				valueNames: []string{"sync_value"},
			},
		}

		resp, err := flags.SyncRequest()
		if err != nil {
			t.Fatalf("SyncRequest failed: %v", err)
		}

		if resp.Version != 3 {
			t.Errorf("Expected version 3, got %d", resp.Version)
		}
	})
}

// Test Sync method
func TestSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SyncFlagsResponse{
			Version: 5,
			Flags: []FlagResponse{
				{Name: "updated_flag", Enabled: false, Overridden: true},
			},
			Values: []ValueResponse{
				{Name: "updated_value", Enabled: true, Overridden: true, ValueOverride: "new_value"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	flags := &FeatureFlags{
		client:   server.Client(),
		httpAddr: server.URL,
		project:  "test-project",
		logger:   &testLogger{},
		state: State{
			version:    4,
			flagState:  make(map[string]FlagState),
			flagNames:  []string{"updated_flag"},
			valueState: make(map[string]ValueState),
			valueNames: []string{"updated_value"},
		},
	}

	err := flags.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if flags.state.version != 5 {
		t.Errorf("Expected version 5, got %d", flags.state.version)
	}
}

// Test FeatureFlags initialization via MakeClient
func TestMakeClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LoadFlagsResponse{
			Version: 1,
			Flags: []FlagResponse{
				{Name: "init_flag", Enabled: true, Overridden: true},
			},
			Values: []ValueResponse{
				{Name: "init_value", Enabled: true, Overridden: true, ValueOverride: 100.0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	defaults := Defaults{
		Flags: []Flag{
			{Name: "init_flag", Enabled: false},
		},
		Values: []Value{
			{Name: "init_value", Value: 50},
		},
	}

	logger := &testLogger{}

	client, err := MakeClient(
		context.Background(),
		server.URL,
		"test-project",
		defaults,
		WithLogger(logger),
		WithSyncInterval(10*time.Second),
	)
	if err != nil {
		t.Fatalf("MakeClient failed: %v", err)
	}

	// Verify client was initialized
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Verify defaults were set correctly
	if !client.Get("init_flag") {
		t.Error("Expected init_flag to be enabled (overridden by server)")
	}

	// Verify sync interval
	if client.syncInterval != 10*time.Second {
		t.Errorf("Expected sync interval 10s, got %v", client.syncInterval)
	}
}

// Test State.Update preserves defaults
func TestStateUpdate(t *testing.T) {
	state := State{
		version: 1,
		flagState: map[string]FlagState{
			"test_flag": {Name: "test_flag", Enabled: false},
		},
		valueState: map[string]ValueState{
			"test_value": {Name: "test_value", Value: 10, DefaultValue: 10, IsOverridden: false},
		},
	}

	// Update with new values from server
	state.Update(2, []FlagResponse{
		{Name: "test_flag", Enabled: true, Overridden: true},
	}, []ValueResponse{
		{Name: "test_value", Enabled: true, Overridden: true, ValueOverride: 20.0, ValueDefault: 10},
	})

	// Verify version updated
	if state.version != 2 {
		t.Errorf("Expected version 2, got %d", state.version)
	}

	// Verify flag updated - use getFlagState to check evaluation
	if !state.getFlagState("test_flag", nil) {
		t.Error("Expected test_flag to be enabled")
	}

	// Verify value updated and default preserved
	valueState := state.valueState["test_value"]
	if valueState.Value != 20.0 {
		t.Errorf("Expected value 20.0, got %v", valueState.Value)
	}
	if valueState.DefaultValue != 10 {
		t.Errorf("Expected default to be preserved as 10, got %v", valueState.DefaultValue)
	}
	if !valueState.IsOverridden {
		t.Error("Expected IsOverridden to be true")
	}
}

// Test State.Update with conditions
func TestStateUpdateWithConditions(t *testing.T) {
	state := State{
		version: 1,
		flagState: map[string]FlagState{
			"conditional_flag": {Name: "conditional_flag", Enabled: false},
		},
		valueState: map[string]ValueState{},
	}

	// Update with a flag that has conditions
	state.Update(2, []FlagResponse{
		{
			Name:       "conditional_flag",
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
		},
	}, []ValueResponse{})

	// Verify flag is enabled for matching context
	ctx := map[string]any{"user.id": float64(123)}
	if !state.getFlagState("conditional_flag", ctx) {
		t.Error("Expected conditional_flag to be true for user.id=123")
	}

	// Verify flag is disabled for non-matching context
	ctx = map[string]any{"user.id": float64(456)}
	if state.getFlagState("conditional_flag", ctx) {
		t.Error("Expected conditional_flag to be false for user.id=456")
	}
}

func TestMustGetValueStringUsesServerDefaultForNewValue(t *testing.T) {
	flags := FeatureFlags{
		logger: &defaultLogger{},
		state: State{
			version:    1,
			flagState:  map[string]FlagState{},
			valueState: map[string]ValueState{},
		},
	}

	flags.state.Update(2, []FlagResponse{}, []ValueResponse{
		{
			Name:          "new_value",
			Enabled:       true,
			Overridden:    false,
			ValueOverride: 123,
			ValueDefault:  "fallback",
		},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	val := flags.MustGetValueString("new_value")
	if val != "fallback" {
		t.Fatalf("expected fallback default, got %v", val)
	}
}
