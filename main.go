// Command tok counts tokens in a text file (or stdin) for a given LLM.
//
// Usage:
//
//	tok <provider>/<model> <file|->
//
// On success: a single integer on stdout.
// On failure: "error: <message>" on stderr and exit code 1.
package main

import (
	"fmt"
	"os"

	"github.com/antegral/tok/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
