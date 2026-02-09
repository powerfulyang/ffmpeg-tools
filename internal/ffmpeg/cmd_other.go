//go:build !windows
// +build !windows

package ffmpeg

import "os/exec"

// hideWindow 在非 Windows 平台上不需要隐藏窗口
func hideWindow(cmd *exec.Cmd) {
	// 非 Windows 平台不需要特殊处理
}
