//go:build windows
// +build windows

package ffmpeg

import (
	"fmt"
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

// killProcessTree 在 Windows 上使用 taskkill 杀死进程树
// Process.Kill() 在 Windows 上只杀死目标进程，不会杀死子进程
// 使用 taskkill /T /F 可以终止整个进程树
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// taskkill /T = 终止子进程树, /F = 强制终止
	kill := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = kill.Run()
}
