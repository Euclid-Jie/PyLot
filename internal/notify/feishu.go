package notify

import (
	"os/exec"
	"syscall"
)

var statusLabels = map[string]string{
	"success": "成功", "error": "失败", "timeout": "超时", "killed": "已停止",
}

// StatusLabel returns a Chinese label for a run status string.
func StatusLabel(status string) string { return statusLabels[status] }

// Feishu sends a message via lark-cli. No-op if either arg is empty.
func Feishu(cliPath, openID, text string) {
	if cliPath == "" || openID == "" {
		return
	}
	cmd := exec.Command(cliPath, "im", "+messages-send", "--as", "bot", "--user-id", openID, "--text", text)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}
