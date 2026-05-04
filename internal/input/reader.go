// Package input handles reading the second CLI argument: either a file path
// or "-" for stdin.
package input

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// binaryDetectWindow is the number of bytes scanned for NUL bytes to detect
// binary input. Matches the convention used by grep, file(1), git, and less.
const binaryDetectWindow = 8192

// isBinary reports whether b appears to be binary by scanning the first
// binaryDetectWindow bytes for a NUL byte (\x00). Plain UTF-8 text never
// contains NUL, so this is a near-perfect discriminator. UTF-16/UTF-32 text
// also trips this check intentionally — the tokenizers cannot handle it
// correctly and the user must convert to UTF-8 first.
func isBinary(b []byte) bool {
	n := len(b)
	if n > binaryDetectWindow {
		n = binaryDetectWindow
	}
	return bytes.IndexByte(b[:n], 0) >= 0
}

// Read returns the full text of the input source. If arg is "-", reads from
// stdin; otherwise reads the file at arg. UTF-8 is assumed; no BOM stripping.
// Returns an error if the input appears to be binary (contains a NUL byte in
// the first 8 KB).
func Read(arg string) (string, error) {
	if arg == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		if isBinary(b) {
			return "", fmt.Errorf("input %q appears to be binary, not text (UTF-8 only — convert UTF-16/UTF-32 first)", "(stdin)")
		}
		return string(b), nil
	}
	b, err := os.ReadFile(arg)
	if err != nil {
		return "", err
	}
	if isBinary(b) {
		return "", fmt.Errorf("input %q appears to be binary, not text (UTF-8 only — convert UTF-16/UTF-32 first)", arg)
	}
	return string(b), nil
}
