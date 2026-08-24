package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForStartupBlocksUntilReady(t *testing.T) {
	app := NewApp()
	result := make(chan error, 1)
	go func() {
		result <- app.waitForStartup()
	}()

	select {
	case <-result:
		t.Fatal("waitForStartup returned before startup completed")
	case <-time.After(20 * time.Millisecond):
	}

	close(app.startupDone)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("waitForStartup() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForStartup did not return after startup completed")
	}
}

func TestWaitForStartupReturnsInitializationError(t *testing.T) {
	app := NewApp()
	want := errors.New("database unavailable")
	app.startupErr = want
	close(app.startupDone)

	if err := app.waitForStartup(); !errors.Is(err, want) {
		t.Fatalf("waitForStartup() error = %v, want wrapped %v", err, want)
	}
}

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
