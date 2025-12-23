package main

import (
	"context"
	"flag"
	"log"
	"time"

	featureflags "github.com/evo-company/featureflags-go"
)

type (
	Flag     = featureflags.Flag
	Variable = featureflags.Variable
	Defaults = featureflags.Defaults
)

const TypeNumber = featureflags.TypeNumber

var SomeFlag = Flag{"some_flag", false}

var defaults = Defaults{
	Flags: []Flag{
		SomeFlag,
	},
}

var variables = []Variable{
	{Name: "user.id", Type: TypeNumber},
}

func main() {
	var customHost string
	flag.StringVar(&customHost, "host", "", "Custom host for the feature flags service")
	flag.Parse()

	host := "https://flags.example.com"
	if customHost != "" {
		host = customHost
	}

	flags, err := featureflags.MakeClient(
		context.Background(),
		host,
		"test.test",
		defaults,
		featureflags.WithVariables(variables),
		featureflags.WithSyncInterval(10*time.Second),
		featureflags.WithLogger(log.Default()),
	)
	if err != nil {
		panic(err.Error())
	}
	// Context with user information for condition evaluation
	ctx := map[string]any{
		"user.id": 123,
	}
	log.Printf("TEST_FLAG: %v", flags.Get("TEST_FLAG", ctx))
}
