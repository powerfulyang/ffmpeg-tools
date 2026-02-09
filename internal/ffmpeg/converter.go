package ffmpeg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Converter handles video conversion using FFmpeg
type Converter struct {
	ffmpegPath  string
	ffprobePath string
	currentCmd  *exec.Cmd
	cancelFunc  context.CancelFunc
}

// NewConverter creates a new Converter instance
func NewConverter() *Converter {
	c := &Converter{}
	c.initPaths()
	return c
}

// getPlatformDir returns the platform-specific directory name
func getPlatformDir() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// initPaths initializes the paths to FFmpeg binaries
func (c *Converter) initPaths() {
	// Determine binary names based on OS
	ffmpegBin := "ffmpeg"
	ffprobeBin := "ffprobe"
	if runtime.GOOS == "windows" {
		ffmpegBin = "ffmpeg.exe"
		ffprobeBin = "ffprobe.exe"
	}

	// 1. 优先检查环境变量中的 ffmpeg
	if envPath, err := exec.LookPath(ffmpegBin); err == nil {
		c.ffmpegPath = envPath
		if probePath, err := exec.LookPath(ffprobeBin); err == nil {
			c.ffprobePath = probePath
		} else {
			c.ffprobePath = ffprobeBin
		}
		return
	}

	// Get the executable directory
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// Platform-specific directory (e.g., windows-amd64, darwin-arm64)
	platformDir := getPlatformDir()

	// 2. Check in resources/ffmpeg/<platform> directory (development)
	devPath := filepath.Join(execDir, "resources", "ffmpeg", platformDir, ffmpegBin)
	if _, err := os.Stat(devPath); err == nil {
		c.ffmpegPath = devPath
		c.ffprobePath = filepath.Join(execDir, "resources", "ffmpeg", platformDir, ffprobeBin)
		return
	}

	// 3. Check in resources/ffmpeg directory without platform (fallback for development)
	devFallback := filepath.Join(execDir, "resources", "ffmpeg", ffmpegBin)
	if _, err := os.Stat(devFallback); err == nil {
		c.ffmpegPath = devFallback
		c.ffprobePath = filepath.Join(execDir, "resources", "ffmpeg", ffprobeBin)
		return
	}

	// 4. Check in same directory as executable (production build)
	prodPath := filepath.Join(execDir, ffmpegBin)
	if _, err := os.Stat(prodPath); err == nil {
		c.ffmpegPath = prodPath
		c.ffprobePath = filepath.Join(execDir, ffprobeBin)
		return
	}

	// 5. Fallback: 需要从 npm mirror 下载
	c.ffmpegPath = ffmpegBin
	c.ffprobePath = ffprobeBin
}

// GetFFmpegPath returns the path to FFmpeg binary
func (c *Converter) GetFFmpegPath() string {
	return c.ffmpegPath
}

// ConvertMOVToVP9WebM converts a MOV file to VP9 WebM with alpha channel
// quality: 0-63, lower is better quality (recommended: 15-35)
func (c *Converter) ConvertMOVToVP9WebM(inputPath, outputPath string, quality int, progressCallback func(float64)) error {
	return c.ConvertMOVToVP9WebMWithContext(context.Background(), inputPath, outputPath, quality, progressCallback)
}

// ConvertMOVToVP9WebMWithContext converts a MOV file to VP9 WebM with alpha channel (with context for cancellation)
func (c *Converter) ConvertMOVToVP9WebMWithContext(ctx context.Context, inputPath, outputPath string, quality int, progressCallback func(float64)) error {
	// Validate quality range
	if quality < 0 {
		quality = 0
	}
	if quality > 63 {
		quality = 63
	}

	// Get video duration for progress calculation
	duration, err := c.getVideoDuration(inputPath)
	if err != nil {
		duration = 0 // Continue without progress reporting
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel
	defer func() {
		c.cancelFunc = nil
		c.currentCmd = nil
	}()

	// Build FFmpeg command for VP9 with alpha channel
	args := []string{
		"-i", inputPath,
		"-c:v", "libvpx-vp9",
		// 1. 处理 Alpha 预乘问题，通常能解决透明边缘的灰边或杂色
		"-vf", "premultiply=inplace=1", 
		"-pix_fmt", "yuva420p",
		// 2. 强制全量程颜色，防止透明背景变深灰
		"-color_range", "pc", 
		"-crf", strconv.Itoa(quality),
		"-b:v", "0",
		"-auto-alt-ref", "0",
		// 3. 某些环境需要 alpha_mode 元数据
		"-metadata:s:v:0", "alpha_mode=1", 
		"-an",
		"-progress", "pipe:1",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	hideWindow(cmd)
	c.currentCmd = cmd

	// Capture stdout for progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Parse progress from stdout
	scanner := bufio.NewScanner(stdout)
	timeRegex := regexp.MustCompile(`out_time_ms=(\d+)`)

	for scanner.Scan() {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			// Clean up partial output file
			os.Remove(outputPath)
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			if timeMs, err := strconv.ParseFloat(matches[1], 64); err == nil && duration > 0 {
				progress := (timeMs / 1000000) / duration * 100
				if progress > 100 {
					progress = 100
				}
				if progressCallback != nil {
					progressCallback(progress)
				}
			}
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Check if it was cancelled
		if ctx.Err() != nil {
			os.Remove(outputPath)
			return fmt.Errorf("转换已取消")
		}
		return fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	// Final progress update
	if progressCallback != nil {
		progressCallback(100)
	}

	return nil
}

// Cancel stops the current conversion
func (c *Converter) Cancel() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	if c.currentCmd != nil && c.currentCmd.Process != nil {
		c.currentCmd.Process.Kill()
	}
}

// getVideoDuration returns the duration of a video file in seconds
func (c *Converter) getVideoDuration(inputPath string) (float64, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		inputPath,
	}

	cmd := exec.Command(c.ffprobePath, args...)
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// GetVideoInfo returns detailed information about a video file
func (c *Converter) GetVideoInfo(inputPath string) (map[string]interface{}, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.Command(c.ffprobePath, args...)
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	// Extract useful information
	info := make(map[string]interface{})

	if format, ok := result["format"].(map[string]interface{}); ok {
		info["filename"] = format["filename"]
		info["duration"] = format["duration"]
		info["size"] = format["size"]
		info["bitrate"] = format["bit_rate"]
	}

	if streams, ok := result["streams"].([]interface{}); ok {
		for _, s := range streams {
			stream := s.(map[string]interface{})
			if stream["codec_type"] == "video" {
				info["width"] = stream["width"]
				info["height"] = stream["height"]
				info["codec"] = stream["codec_name"]
				info["fps"] = stream["r_frame_rate"]
				info["pixelFormat"] = stream["pix_fmt"]

				// Check if video has alpha channel
				pixFmt := fmt.Sprintf("%v", stream["pix_fmt"])
				info["hasAlpha"] = strings.Contains(pixFmt, "a") || strings.Contains(pixFmt, "rgba") || strings.Contains(pixFmt, "yuva")
				break
			}
		}
	}

	return info, nil
}

// CheckFFmpegInstalled checks if FFmpeg is available
func (c *Converter) CheckFFmpegInstalled() bool {
	log.Printf("[FFmpeg] CheckFFmpegInstalled 开始, path=%s", c.ffmpegPath)
	cmd := exec.Command(c.ffmpegPath, "-version")
	hideWindow(cmd)
	log.Println("[FFmpeg] 执行 ffmpeg -version...")
	// 使用 CombinedOutput 替代 Run，避免在 Windows 隐藏窗口模式下卡住
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[FFmpeg] ffmpeg -version 失败: %v", err)
	} else {
		log.Printf("[FFmpeg] ffmpeg -version 成功, output=%s", string(output[:min(len(output), 100)]))
	}
	return err == nil
}
