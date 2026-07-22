package processutil

import (
	"fmt"
	"os/exec"
	"syscall"
)

// KillTree terminates a Windows process tree without creating a visible console window.
func KillTree(pid int) error {
	return taskkillTreeCommand(pid).Run()
}

func taskkillTreeCommand(pid int) *exec.Cmd {
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
