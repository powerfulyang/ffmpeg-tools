//go:build windows
// +build windows

package ffmpeg

import (
	"os"
	"os/exec"
	"syscall"
)

// hideWindow 设置 Windows 下隐藏命令行窗口
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	// 确保 stdin 被设置，避免命令因等待 stdin 而卡住
	if cmd.Stdin == nil {
		cmd.Stdin, _ = os.Open(os.DevNull)
	}
}
