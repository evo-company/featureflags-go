FeatureFlags client library for Go language

See ``examples`` directory for complete examples

[Featureflags server repository](https://github.com/evo-company/featureflags)
[Featureflags documentation](https://featureflags.readthedocs.io/en/latest/)

# Usage

### Installation

To install the package, use `go get`:

```bash
go get github.com/evo-company/featureflags-go
```

### Usage

The client uses the functional options pattern for flexible configuration. The `httpAddr`, `project`, and `defaults` parameters are required, while other options are optional.

#### Available Options

- `WithVariables(variables []Variable)` - Set variables for targeting rules
- `WithSyncInterval(interval time.Duration)` - Set sync interval (default: 10 seconds)
- `WithLogger(logger Logger)` - Set a custom logger (default: no-op logger)

#### Working with Values

Values allow you to store configuration settings (strings, integers, etc.) that can be overridden by the server.

**Default Values**: When you initialize the client, you provide default values:
```go
defaults := featureflags.Defaults{
    Values: []featureflags.Value{
        {Name: "http_timeout", Value: 30},      // Default timeout: 30 seconds
        {Name: "api_endpoint", Value: "https://api.example.com"},
    },
}
```

**Server Overrides**: The server can override these defaults. For example, it might change `http_timeout` from 30 to 50.

**Retrieving Values**: Two approaches for type-safe value retrieval:

**1. Error-returning getters** (recommended when you need to handle failures):
- `GetValueInt(name string) (int, error)` - Returns error if not found or wrong type
- `GetValueString(name string) (string, error)` - Returns error if not found or wrong type

**2. Must getters** (recommended when you want guaranteed defaults):
- `MustGetValueInt(name string) int` - Returns value or default, panics if key never defined
- `MustGetValueString(name string) string` - Returns value or default, panics if key never defined

**Other methods**:
- `GetValue(name string) interface{}` - Returns raw value (requires manual type casting)
- `IsValueOverridden(name string) bool` - Check if server overrode the default

**Safety guarantees**:
- `GetValue*` methods return errors instead of zero values (preventing dangerous defaults like 0 timeout)
- `MustGetValue*` methods guarantee a value is returned (either current or default)
- `MustGetValue*` panics only on programming errors (requesting undefined keys)
- Type mismatches are logged and fall back to defaults in Must* versions

#### Quick Start

Minimal example with only required parameters:

```go
defaults := featureflags.Defaults{
    Flags: []featureflags.Flag{
        {Name: "MY_FLAG", Enabled: false},
    },
}

client, err := featureflags.MakeClient(
    context.Background(),
    "http://your-flags-service",
    "my-project",
    defaults,
)
```

#### Comprehensive Example

Here's a comprehensive example demonstrating how to initialize the client, define defaults and variables, and retrieve flag and value states:

```go
package main

import (
 "context"
 "log"
 "time"

 featureflags "github.com/evo-company/featureflags-go"
)

func main() {
 // 1. Initialize Defaults for flags and values
 defaults := featureflags.Defaults{
  Flags: []featureflags.Flag{
   {Name: "TEST_FLAG_ENABLED_BY_DEFAULT", Enabled: true},
   {Name: "TEST_FLAG_DISABLED_BY_DEFAULT", Enabled: false},
  },
  Values: []featureflags.Value{
   {Name: "WELCOME_MESSAGE", Value: "Hello, Gophers!"},
   {Name: "MAX_USERS", Value: 100},
  },
 }

 // 2. Define Variables (e.g., for targeting rules)
 // These would typically be passed to the FeatureFlags backend for evaluation.
 variables := []featureflags.Variable{
  {Name: "user_id", Type: featureflags.TypeString},
  {Name: "country", Type: featureflags.TypeString},
 }

 // 3. Initialize a FeatureFlags Client
 // Replace "http://your-flags-service" with the actual address of your feature flags backend.

  // MakeClient will make a http call to featureflags server api in order to
  // initialize project and sync flags/values states
  // Also it will run a goroutine to call Sync every `syncInterval`
 flagsClient, err := featureflags.MakeClient(
  context.Background(),
  "http://your-flags-service", // Your HTTP feature flags service address
  "my-project",               // Your project name
  defaults,                   // Default flags and values
  featureflags.WithVariables(variables),
  featureflags.WithSyncInterval(10*time.Second),
  featureflags.WithLogger(log.Default()),
 )
 if err != nil {
  log.Fatalf("Failed to create feature flags client: %v", err)
 }

 // 4. Get flag state
 if flagsClient.Get("TEST_FLAG_ENABLED_BY_DEFAULT") {
  println("TEST_FLAG_ENABLED_BY_DEFAULT is enabled!")
 } else {
  println("TEST_FLAG_ENABLED_BY_DEFAULT is disabled.")
 }

 if flagsClient.Get("NON_EXISTENT_FLAG") {
  println("NON_EXISTENT_FLAG is enabled (this shouldn't happen unless defaulted elsewhere).")
 } else {
  println("NON_EXISTENT_FLAG is disabled (as expected).")
 }

 // 5. Get value state

 // Option 1: Must getters - guaranteed to return a value (current or default)
 // Best for production code where you want safe fallbacks
 welcomeMessage := flagsClient.MustGetValueString("WELCOME_MESSAGE")
 println("Welcome Message:", welcomeMessage)

 maxUsers := flagsClient.MustGetValueInt("MAX_USERS")
 println("Maximum Users:", maxUsers)

 // Option 2: Error-returning getters - handle errors explicitly
 // Best when you need to know if a value fetch failed
 if timeout, err := flagsClient.GetValueInt("HTTP_TIMEOUT"); err != nil {
  log.Printf("Failed to get HTTP_TIMEOUT: %v", err)
 } else {
  println("HTTP Timeout:", timeout)
 }

 // Check if a value was overridden by the server
 if flagsClient.IsValueOverridden("MAX_USERS") {
  println("MAX_USERS was overridden by the server")
 } else {
  println("MAX_USERS is using the default value")
 }

 // Generic getter (returns interface{} - requires manual type casting)
 rawValue := flagsClient.GetValue("WELCOME_MESSAGE")
 if msg, ok := rawValue.(string); ok {
  println("Raw value:", msg)
 }

 // The client will continue to sync flags in a background loop.
 // In a real application, you might want to keep the main goroutine alive
 // or manage the lifecycle of the flagsClient more carefully.
 select {} // Keep main goroutine alive
}
```

## Examples

To run the complete example application:

**Without custom host (uses default <https://flags.example.com>):**

```bash
go run example/main.go
```

**With custom host:**

```bash
go run example/main.go -host http://localhost:5000
```
