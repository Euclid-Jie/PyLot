package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := existingDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("existingDirectory() = %q, want %q", got, want)
	}
}

func TestExistingDirectoryRejectsInvalidPath(t *testing.T) {
	if _, err := existingDirectory(""); err == nil {
		t.Fatal("empty directory returned no error")
	}
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := existingDirectory(file); err == nil {
		t.Fatal("file path returned no error")
	}
}
