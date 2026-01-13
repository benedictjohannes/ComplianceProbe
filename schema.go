//go:build builder

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
)

func generateSchema() {
	reflector := &jsonschema.Reflector{
		DoNotReference: false,
		ExpandedStruct: true,
	}

	s := reflector.Reflect(&ReportConfig{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to generate schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
