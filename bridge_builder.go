//go:build builder

package main

func callGenerateSchema() {
	generateSchema()
}

func callRunPreprocess(input, output string) {
	runPreprocess(input, output)
}
