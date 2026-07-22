package processutil

import (
	"reflect"
	"testing"
)

func TestTaskkillTreeCommandIsHidden(t *testing.T) {
	cmd := taskkillTreeCommand(1234)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("taskkill command must hide its console window")
	}
	wantArgs := []string{"taskkill", "/F", "/T", "/PID", "1234"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, wantArgs)
	}
}
