package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/internal/elevation"
	"github.com/benedictjohannes/crobe/internal/headerflags"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("crobe", flag.ContinueOnError)
	folderFlag := flags.String("folder", "", "Folder to write reports to (default \"reports\")")
	workerFlag := flags.String("worker", "", "Run in elevated worker mode connected to socket URI")
	var headersFlags headerflags.HeaderFlags
	flags.Var(&headersFlags, "H", "Custom header for remote playbook fetching (eg: 'Authorization: Bearer <TOKEN>'). Specify multiple times for each header you want to add.")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	if *workerFlag != "" {
		if err := elevation.RunWorker(*workerFlag); err != nil {
			fmt.Printf("❌ Worker Error: %v\n", err)
			return 1
		}
		return 0
	}

	headers := headersFlags.ToMap()

	reportwriter.DefaultReportsDir = *folderFlag

	configPath := flags.Arg(0)
	if configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]'")
		return 1
	}

	config, _, err := configsource.LoadConfig(configPath, headers)
	if err != nil {
		fmt.Printf("❌ Failed to load playbook %s: %v\n", configPath, err)
		return 1
	}

	// TODO this line to below are not covered by tests

	// Validate as Agent
	if err := playbook.ValidateConfig(*config, true); err != nil {
		fmt.Printf("❌ Validation Error: %v\n", err)
		return 1
	}

	cleanup, err := elevation.SetupElevation(config)
	if err != nil {
		fmt.Printf("❌ Elevation Setup Error: %v\n", err)
		return 1
	}
	defer cleanup()

	trace := director.Run(context.Background(), *config)
	result := report.GenerateReport(trace)
	if err := reportwriter.DispatchReport(config, result); err != nil {
		fmt.Printf("❌ Reporting Error: %v\n", err)
		return 1
	}

	if result.Structured.Stats.Failed > 0 {
		return 1
	}

	return 0
}
