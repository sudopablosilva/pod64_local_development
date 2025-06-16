package main

import (
	"flag"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/sudopablosilva/poc_bdd/integration-tests/steps"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can be changed to "pretty"
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	steps.InitializeJobProcessingSteps(ctx)
	steps.InitializeServiceCommunicationSteps(ctx)
	steps.InitializeInfrastructureSteps(ctx)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opts.Paths = flag.Args()

	status := godog.TestSuite{
		Name:                "poc_bdd",
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}.Run()

	// Optional: to print more details, change it to STDOUT
	// fmt.Fprintf(os.Stderr, "Tests finished with status: %d\n", status)

	if len(opts.Paths) == 0 {
		opts.Paths = []string{"features"}
		status = godog.TestSuite{
			Name:                "poc_bdd",
			ScenarioInitializer: InitializeScenario,
			Options:             &opts,
		}.Run()
	}

	os.Exit(status)
}
