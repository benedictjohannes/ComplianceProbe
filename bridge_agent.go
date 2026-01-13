//go:build !builder

package main

import (
	"fmt"
	"os"
)

func callGenerateSchema() {
	fmt.Println("❌ Error: --schema requires the ComplianceProbe Builder binary.")
	fmt.Println("Tip: Build with -tags builder to enable builder features.")
	os.Exit(1)
}

func callRunPreprocess(_, _ string) {
	fmt.Println("❌ Error: --preprocess requires the ComplianceProbe Builder binary.")
	fmt.Println("Tip: Build with -tags builder to enable builder features.")
	os.Exit(1)
}
