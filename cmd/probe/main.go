package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/internal/configsource"
	"github.com/benedictjohannes/crobe/internal/desktop"
	"github.com/benedictjohannes/crobe/internal/elevation"
	"github.com/benedictjohannes/crobe/internal/headerflags"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/internal/server"
	"github.com/benedictjohannes/crobe/report"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("crobe", flag.ContinueOnError)
	folderFlag := flags.String("folder", "", "Folder to write reports to (default \"reports\")")
	workerFlag := flags.String("worker", "", "Run in elevated worker mode connected to socket URI")
	uiFlag := flags.Bool("ui", false, "Start embedded Web UI server")
	hostFlag := flags.String("host", "127.0.0.1", "Host address to bind Web UI to")
	portFlag := flags.Int("port", 0, "Port number to bind Web UI to (default: OS assigned / PORT env)")
	noOpenFlag := flags.Bool("no-open", false, "Disable automatic browser launch on Web UI startup")
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
	configPath := flags.Arg(0)

	// Determine if UI mode should be enabled
	isUI := *uiFlag
	if !isUI && configPath == "" && len(args) == 0 {
		if desktop.IsDesktopGUI() {
			isUI = true
		} else {
			fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]' or run 'crobe --ui'")
			return 1
		}
	} else if !isUI && configPath == "" {
		fmt.Println("❌ Error: No playbook provided. Use 'crobe [path/to/playbook.yaml]' or run 'crobe --ui'")
		return 1
	}

	if isUI {
		port := *portFlag
		if port == 0 {
			if portEnv := os.Getenv("PORT"); portEnv != "" {
				if p, err := strconv.Atoi(portEnv); err == nil && p > 0 {
					port = p
				}
			}
		}

		cfg := server.Config{
			Host:      *hostFlag,
			Port:      port,
			CLIFolder: *folderFlag,
			NoOpen:    *noOpenFlag,
		}

		srv, err := server.NewServer(cfg)
		if err != nil {
			fmt.Printf("❌ Failed to initialize Web UI server: %v\n", err)
			return 1
		}

		// Preload playbook if provided on CLI
		if configPath != "" {
			config, rawBytes, err := configsource.LoadConfig(configPath, headers)
			if err != nil {
				srv.StateManager().SetLoadError(server.ErrCodePlaybookParseFailed, fmt.Sprintf("Failed to load playbook %s: %v", configPath, err), nil)
			} else {
				valErrors := config.Validate(false)
				srv.StateManager().SetPlaybook(config, rawBytes, valErrors)
			}
		}

		if err := srv.Start(); err != nil {
			fmt.Printf("❌ Failed to start Web UI server: %v\n", err)
			return 1
		}

		fmt.Println("🌐 Compliance Probe UI is running at:")
		fmt.Printf("   %s\n\n", srv.URL())

		if !*noOpenFlag && desktop.IsDesktopGUI() {
			_ = desktop.OpenBrowser(srv.URL())
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		select {
		case <-sigChan:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		case <-srv.ShutdownChan():
		}

		return 0
	}

	// Headless CLI execution
	reportwriter.DefaultReportsDir = *folderFlag

	config, _, err := configsource.LoadConfig(configPath, headers)
	if err != nil {
		fmt.Printf("❌ Failed to load playbook %s: %v\n", configPath, err)
		return 1
	}

	// Validate as Agent
	if errs := config.Validate(true); len(errs) > 0 {
		fmt.Println("❌ Playbook Validation Failed:")
		for _, err := range errs {
			fmt.Printf("  • %s\n", err.Error())
		}
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
