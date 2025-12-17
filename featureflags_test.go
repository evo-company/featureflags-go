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
					{Name: "test_flag", Enabled: true},
				},
				Values: []ValueResponse{
					{Name: "test_value", Value: "hello"},
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
		if len(resp.Flags) != 1 {
			t.Errorf("Expected 1 flag, got %d", len(resp.Flags))
		}
		if resp.Flags[0].Name != "test_flag" {
			t.Errorf("Expected flag name 'test_flag', got %s", resp.Flags[0].Name)
		}
		if !resp.Flags[0].Enabled {
			t.Error("Expected flag to be enabled")
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
				{Name: "feature_a", Enabled: true},
				{Name: "feature_b", Enabled: false},
			},
			Values: []ValueResponse{
				{Name: "timeout", Value: float64(30)},
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
	if !flags.state.FlagState("feature_a") {
		t.Error("Expected feature_a to be enabled")
	}
	if flags.state.FlagState("feature_b") {
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
					{Name: "sync_flag", Enabled: true},
				},
				Values: []ValueResponse{
					{Name: "sync_value", Value: 42.0},
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
				{Name: "updated_flag", Enabled: false},
			},
			Values: []ValueResponse{
				{Name: "updated_value", Value: "new_value"},
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
				{Name: "init_flag", Enabled: true},
			},
			Values: []ValueResponse{
				{Name: "init_value", Value: 100.0},
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
		val := flags.GetValue("string_value")
		if strVal, ok := val.(string); !ok || strVal != "hello" {
			t.Errorf("Expected 'hello', got %v", val)
		}
	})

	t.Run("get int value", func(t *testing.T) {
		val := flags.GetValue("int_value")
		if intVal, ok := val.(int); !ok || intVal != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})

	t.Run("get non-existent value", func(t *testing.T) {
		val := flags.GetValue("non_existent")
		if val != nil {
			t.Errorf("Expected nil, got %v", val)
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
		val, err := flags.GetValueInt("int_value")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})

	t.Run("GetValueInt - success with float64", func(t *testing.T) {
		val, err := flags.GetValueInt("float_value")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != 3 {
			t.Errorf("Expected 3, got %d", val)
		}
	})

	t.Run("GetValueInt - error on wrong type", func(t *testing.T) {
		_, err := flags.GetValueInt("string_value")
		if err == nil {
			t.Error("Expected error for wrong type")
		}
	})

	t.Run("GetValueInt - error on non-existent", func(t *testing.T) {
		_, err := flags.GetValueInt("non_existent")
		if err == nil {
			t.Error("Expected error for non-existent value")
		}
	})

	t.Run("GetValueString - success", func(t *testing.T) {
		val, err := flags.GetValueString("string_value")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("Expected 'hello', got %s", val)
		}
	})

	t.Run("GetValueString - error on wrong type", func(t *testing.T) {
		_, err := flags.GetValueString("int_value")
		if err == nil {
			t.Error("Expected error for wrong type")
		}
	})

	t.Run("GetValueString - error on non-existent", func(t *testing.T) {
		_, err := flags.GetValueString("non_existent")
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
		val := flags.MustGetValueInt("int_value")
		if val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}
	})

	t.Run("MustGetValueInt - success with float64", func(t *testing.T) {
		val := flags.MustGetValueInt("float_value")
		if val != 3 {
			t.Errorf("Expected 3, got %d", val)
		}
	})

	t.Run("MustGetValueInt - fallback to default on wrong type", func(t *testing.T) {
		val := flags.MustGetValueInt("wrong_type_int")
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
		flags.MustGetValueInt("non_existent")
	})

	t.Run("MustGetValueString - success", func(t *testing.T) {
		val := flags.MustGetValueString("string_value")
		if val != "hello" {
			t.Errorf("Expected 'hello', got %s", val)
		}
	})

	t.Run("MustGetValueString - fallback to default on wrong type", func(t *testing.T) {
		val := flags.MustGetValueString("wrong_type_str")
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
		flags.MustGetValueString("non_existent")
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
		{Name: "test_flag", Enabled: true},
	}, []ValueResponse{
		{Name: "test_value", Value: 20.0},
	})

	// Verify version updated
	if state.version != 2 {
		t.Errorf("Expected version 2, got %d", state.version)
	}

	// Verify flag updated
	if !state.flagState["test_flag"].Enabled {
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
