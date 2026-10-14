package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesFiles(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join("..", "..", "testdata", "Go-Regular.ttf")
	output := filepath.Join(dir, "sample")
	if err := run(fontPath, output, "A ", "", 16, 256); err != nil {
		t.Fatal(err)
	}
	descriptor, err := os.ReadFile(output + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	atlas, err := os.ReadFile(output + "_0.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(atlas) == 0 || !strings.Contains(string(descriptor), `file="sample_0.png"`) || !strings.Contains(string(descriptor), "chars count=2") {
		t.Fatalf("unexpected CLI outputs: descriptor %s, PNG bytes %d", descriptor, len(atlas))
	}
}

func TestRunDefaultsToPrintableASCII(t *testing.T) {
	dir := t.TempDir()
	fontPath := filepath.Join("..", "..", "testdata", "Go-Regular.ttf")
	output := filepath.Join(dir, "default")
	if err := run(fontPath, output, "", "", 12, 1024); err != nil {
		t.Fatal(err)
	}
	descriptor, err := os.ReadFile(output + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(descriptor), "chars count=95") {
		t.Fatalf("CLI default did not generate printable ASCII: %s", descriptor)
	}
}

func TestRunRequiresFontAndRejectsBothCharacterInputs(t *testing.T) {
	if err := run("", "out", "", "", 16, 256); err == nil {
		t.Fatal("expected missing font error")
	}
	if err := run("font.ttf", "out", "A", "chars.txt", 16, 256); err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("got error %v; want mutually-exclusive character input error", err)
	}
}
