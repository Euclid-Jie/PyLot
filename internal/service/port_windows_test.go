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

func TestProcessBelongsToTree(t *testing.T) {
	processes := map[int]processSnapshotEntry{
		20008: {parentPID: 100},
		30000: {parentPID: 20008},
		55660: {parentPID: 30000},
		99999: {parentPID: 100},
	}
	if !processBelongsToTree(55660, 20008, processes) {
		t.Fatal("listener child was not associated with its service root")
	}
	if processBelongsToTree(99999, 20008, processes) {
		t.Fatal("unrelated process was associated with the service root")
	}
	root := findProcessTreeRoot(55660, map[int]struct{}{100: {}, 20008: {}}, processes)
	if root != 20008 {
		t.Fatalf("matched root PID = %d, want 20008", root)
	}
}

func TestProcessBelongsToTreeHandlesCycles(t *testing.T) {
	processes := map[int]processSnapshotEntry{
		10: {parentPID: 11},
		11: {parentPID: 10},
	}
	if processBelongsToTree(10, 20, processes) {
		t.Fatal("cyclic parent chain matched an unrelated root")
	}
}
