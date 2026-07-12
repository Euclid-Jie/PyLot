package service

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestFindPortOwner(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	deadline := time.Now().Add(2 * time.Second)
	for {
		owner, err := FindPortOwner(port)
		if err != nil {
			t.Fatal(err)
		}
		if owner != nil {
			if owner.PID != os.Getpid() {
				t.Fatalf("port %d owner PID = %d, want %d", port, owner.PID, os.Getpid())
			}
			if owner.ProcessName == "" {
				t.Fatal("process name is empty")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("listener on port %d was not found", port)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestFindPortOwnerRejectsInvalidPort(t *testing.T) {
	for _, port := range []int{-1, 0, 65536} {
		if _, err := FindPortOwner(port); err == nil {
			t.Fatalf("FindPortOwner(%d) returned no error", port)
		}
	}
}
