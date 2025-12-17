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
  defaults,
  variables,
  10*time.Second,             // Sync interval as time.Duration
  log.Default(),              // Logger
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
 welcomeMessage := flagsClient.GetValue("WELCOME_MESSAGE")
 if msg, ok := welcomeMessage.(string); ok {
  println("Welcome Message:", msg)
 } else {
  println("Welcome Message not found or not a string.")
 }

 maxUsers := flagsClient.GetValue("MAX_USERS")
 if count, ok := maxUsers.(int); ok {
  println("Maximum Users:", count)
 } else {
  println("Maximum Users not found or not an integer.")
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
