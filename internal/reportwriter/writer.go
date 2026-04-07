package reportwriter

import (
	"compliance-probe/report"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func WriteToFolder(res report.FinalResult) {
	now := time.Now()
	timestamp := now.Format("060102-150405")

	reportsDir := "reports"
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		os.MkdirAll(reportsDir, 0755)
	}

	reportBase := filepath.Join(reportsDir, timestamp+".report")
	logFile := reportBase + ".log"
	mdFile := reportBase + ".md"
	jsonFile := reportBase + ".json"

	os.WriteFile(logFile, []byte(res.Log), 0644)
	os.WriteFile(mdFile, []byte(res.Markdown), 0644)
	jsonBytes, _ := json.MarshalIndent(res.Structured, "", "  ")
	os.WriteFile(jsonFile, jsonBytes, 0644)

	fmt.Printf("\n✅ Generation Complete!\n")
	fmt.Printf("📊 PASS: %d, FAIL: %d\n", res.Structured.Stats.Passed, res.Structured.Stats.Failed)
	fmt.Printf("📝 Log: %s\n", logFile)
	fmt.Printf("📝 Markdown: %s\n", mdFile)
	fmt.Printf("📊 JSON Report: %s\n", jsonFile)
}
