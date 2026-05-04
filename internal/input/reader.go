// Package input handles reading the second CLI argument: either a file path
// or "-" for stdin.
package input

import (
	"io"
	"os"
)

// Read returns the full text of the input source. If arg is "-", reads from
// stdin; otherwise reads the file at arg. UTF-8 is assumed; no BOM stripping.
func Read(arg string) (string, error) {
	if arg == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := os.ReadFile(arg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
