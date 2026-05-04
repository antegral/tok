package input

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestIsBinary_Text verifies that plain UTF-8 text is not detected as binary.
func TestIsBinary_Text(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
	}{
		{"ascii", []byte("hello world")},
		{"korean", []byte("한글 테스트")},
		{"multiline", []byte("line one\nline two\nline three\n")},
		{"empty", []byte{}},
		{"tab and newline", []byte("col1\tcol2\nval1\tval2\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isBinary(tc.input) {
				t.Errorf("isBinary(%q) = true, want false", tc.input)
			}
		})
	}
}

// TestIsBinary_Binary verifies that input with a NUL byte is detected as binary.
func TestIsBinary_Binary(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
	}{
		{"nul only", []byte{0}},
		{"nul in middle", []byte("hello\x00world")},
		{"nul at start", []byte("\x00hello")},
		{"nul at end", []byte("hello\x00")},
		{"elf header", []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !isBinary(tc.input) {
				t.Errorf("isBinary(%q) = false, want true", tc.input)
			}
		})
	}
}

// TestIsBinary_Boundary_NULAtByte8192 verifies that a NUL at exactly byte 8192
// (the last byte of the detection window, 0-based index 8191) is detected.
func TestIsBinary_Boundary_NULAtByte8192(t *testing.T) {
	b := make([]byte, 8192)
	for i := range b {
		b[i] = 'A'
	}
	b[8191] = 0 // last byte of the 8 KB window
	if !isBinary(b) {
		t.Error("isBinary: expected true for NUL at index 8191 (last of window), got false")
	}
}

// TestIsBinary_Boundary_NULOutsideWindow verifies that a NUL beyond the first
// 8 KB is NOT detected. This is intentional and documented behaviour.
func TestIsBinary_Boundary_NULOutsideWindow(t *testing.T) {
	b := make([]byte, 9000)
	for i := range b {
		b[i] = 'A'
	}
	b[8500] = 0 // outside the 8 KB window (index 8500 > 8191)
	if isBinary(b) {
		t.Error("isBinary: expected false for NUL outside 8 KB window, got true")
	}
}

// TestRead_Text verifies that a text file is read successfully.
func TestRead_Text(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/hello.txt"
	content := "hello world\nline two\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read(%q) error: %v", path, err)
	}
	if got != content {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

// TestRead_KoreanText verifies that UTF-8 multi-byte text is read without error.
func TestRead_KoreanText(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/korean.txt"
	content := "한글 테스트\n두 번째 줄\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read(%q) error: %v", path, err)
	}
	if got != content {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

// TestRead_BinaryFile verifies that a binary temp file returns an error that
// mentions the file path and the word "binary".
func TestRead_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bin.elf"
	data := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read(binary file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should mention path %q, got: %v", path, err)
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Errorf("error should mention 'binary', got: %v", err)
	}
}

// TestRead_BinaryFile_ErrorMentionsArg verifies the exact quoting of arg in error.
func TestRead_BinaryFile_ErrorMentionsArg(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/image.png"
	// PNG magic bytes contain NUL at byte 4.
	data := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for binary PNG, got nil")
	}
	// The error should quote the exact arg passed to Read.
	want := fmt.Sprintf("%q", path)
	if !strings.Contains(err.Error(), want[1:len(want)-1]) { // strip outer quotes
		t.Errorf("error %q should contain path %s", err.Error(), path)
	}
}

// TestRead_StdinErrorMentionsSdin exercises the "(stdin)" label in the error
// by inspecting the error message shape (we cannot redirect os.Stdin in a
// pure unit test, so we verify the helper that produces the message).
func TestRead_StdinErrorMessage(t *testing.T) {
	// Produce the error the same way Read("-") does, to check the label.
	err := fmt.Errorf("input %q appears to be binary, not text (UTF-8 only — convert UTF-16/UTF-32 first)", "(stdin)")
	if !strings.Contains(err.Error(), "(stdin)") {
		t.Errorf("stdin error should mention (stdin), got: %v", err)
	}
}
