//go:build !windows
// +build !windows

package ffmpeg

import (
	"os/exec"
	"syscall"
)

// hideWindow 在非 Windows 平台上不需要隐藏窗口
func hideWindow(cmd *exec.Cmd) {
	// 非 Windows 平台不需要特殊处理
}

// killProcessTree 在非 Windows 平台上通过发送信号给进程组来杀死整个进程树
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	// 发送 SIGKILL 给进程组
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
