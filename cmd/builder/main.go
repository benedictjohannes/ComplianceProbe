package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benedictjohannes/crobe/internal/runner"
	"github.com/benedictjohannes/crobe/internal/transpile"
	"github.com/benedictjohannes/crobe/playbook"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var (
		schemaFlag     *bool
		preprocessFlag *bool
		inputFlag      *string
		outputFlag     *string
	)

	return runner.Run(args, runner.Options{
		Name:    "crobe-builder",
		IsAgent: false,
		PreprocessHook: func(config *playbook.Playbook, baseDir string) error {
			return transpile.Preprocess(config, baseDir)
		},
		CustomFlags: func(fs *flag.FlagSet) {
			schemaFlag = fs.Bool("schema", false, "Output the configuration JSON schema and exit")
			preprocessFlag = fs.Bool("preprocess", false, "Preprocess a raw YAML into a baked playbook")
			inputFlag = fs.String("input", "", "Input raw YAML file (for preprocess)")
			outputFlag = fs.String("output", "playbook.yaml", "Output baked YAML file (for preprocess)")
		},
		CustomHandler: func(fs *flag.FlagSet) (bool, int) {
			if schemaFlag != nil && *schemaFlag {
				schema, err := playbook.GenerateSchema()
				if err != nil {
					fmt.Printf("❌ Failed to generate schema: %v\n", err)
					return true, 1
				}
				fmt.Println(schema)
				return true, 0
			}

			if preprocessFlag != nil && *preprocessFlag {
				if inputFlag == nil || *inputFlag == "" {
					fmt.Println("❌ Error: --input is required for --preprocess")
					return true, 1
				}
				outPath := "playbook.yaml"
				if outputFlag != nil && *outputFlag != "" {
					outPath = *outputFlag
				}
				return true, runPreprocess(*inputFlag, outPath)
			}

			return false, 0
		},
	})
}

func runPreprocess(inputPath string, outputPath string) int {
	if err := transpile.BakeFile(inputPath, outputPath); err != nil {
		fmt.Printf("❌ Preprocessing Failed: %v\n", err)
		return 1
	}
	fmt.Printf("🚀 Preprocessing Complete! Baked playbook saved to: %s\n", outputPath)
	return 0
}


