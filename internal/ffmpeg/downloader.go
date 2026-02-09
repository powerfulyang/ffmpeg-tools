package ffmpeg

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// ffmpeg-static 版本
	ffmpegVersion = "b6.1.1"
	// npmmirror CDN 基础 URL
	baseURL = "https://cdn.npmmirror.com/binaries/ffmpeg-static/" + ffmpegVersion
)

// 平台下载信息
type platformInfo struct {
	ffmpegURL  string
	ffprobeURL string
	ext        string
}

// 获取当前平台的下载信息
func getPlatformInfo() (*platformInfo, error) {
	platform := runtime.GOOS + "-" + runtime.GOARCH

	platformMap := map[string]platformInfo{
		"windows-amd64": {
			ffmpegURL:  baseURL + "/ffmpeg-win32-x64.gz",
			ffprobeURL: baseURL + "/ffprobe-win32-x64.gz",
			ext:        ".exe",
		},
		"darwin-amd64": {
			ffmpegURL:  baseURL + "/ffmpeg-darwin-x64.gz",
			ffprobeURL: baseURL + "/ffprobe-darwin-x64.gz",
			ext:        "",
		},
		"darwin-arm64": {
			ffmpegURL:  baseURL + "/ffmpeg-darwin-arm64.gz",
			ffprobeURL: baseURL + "/ffprobe-darwin-arm64.gz",
			ext:        "",
		},
	}

	info, ok := platformMap[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	return &info, nil
}

// EnsureFFmpeg 确保 FFmpeg 已安装，如果没有则自动下载
func EnsureFFmpeg(progressCallback func(status string, progress float64)) error {
	log.Println("[FFmpeg] EnsureFFmpeg 开始执行...")
	c := NewConverter()
	log.Printf("[FFmpeg] Converter 创建完成, ffmpegPath=%s", c.GetFFmpegPath())

	// 检查 FFmpeg 是否已经可用
	log.Println("[FFmpeg] 开始检查 FFmpeg 是否已安装...")
	if c.CheckFFmpegInstalled() {
		log.Println("[FFmpeg] FFmpeg 已安装!")
		if progressCallback != nil {
			progressCallback("FFmpeg 已就绪", 100)
		}
		return nil
	}
	log.Println("[FFmpeg] FFmpeg 未安装，需要下载")

	// 获取平台信息
	info, err := getPlatformInfo()
	if err != nil {
		return err
	}

	// 获取目标目录
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// 使用平台特定目录
	platform := runtime.GOOS + "-" + runtime.GOARCH
	targetDir := filepath.Join(execDir, "resources", "ffmpeg", platform)

	// 创建目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 下载 ffmpeg
	ffmpegPath := filepath.Join(targetDir, "ffmpeg"+info.ext)
	log.Printf("[FFmpeg] 准备下载 ffmpeg 到: %s", ffmpegPath)
	log.Printf("[FFmpeg] 下载 URL: %s", info.ffmpegURL)
	if progressCallback != nil {
		progressCallback("正在下载 ffmpeg...", 10)
	}
	if err := downloadAndExtract(info.ffmpegURL, ffmpegPath, func(p float64) {
		if progressCallback != nil {
			progressCallback("正在下载 ffmpeg...", 10+p*0.4)
		}
	}); err != nil {
		log.Printf("[FFmpeg] 下载 ffmpeg 失败: %v", err)
		return fmt.Errorf("下载 ffmpeg 失败: %w", err)
	}
	log.Println("[FFmpeg] ffmpeg 下载完成")

	// 下载 ffprobe
	ffprobePath := filepath.Join(targetDir, "ffprobe"+info.ext)
	if progressCallback != nil {
		progressCallback("正在下载 ffprobe...", 55)
	}
	if err := downloadAndExtract(info.ffprobeURL, ffprobePath, func(p float64) {
		if progressCallback != nil {
			progressCallback("正在下载 ffprobe...", 55+p*0.4)
		}
	}); err != nil {
		return fmt.Errorf("下载 ffprobe 失败: %w", err)
	}

	// 设置执行权限 (Unix)
	if runtime.GOOS != "windows" {
		os.Chmod(ffmpegPath, 0755)
		os.Chmod(ffprobePath, 0755)
	}

	// 重新初始化路径
	c.initPaths()

	if progressCallback != nil {
		progressCallback("FFmpeg 下载完成", 100)
	}

	return nil
}

// downloadAndExtract 下载并解压 gzip 文件
func downloadAndExtract(url, outputPath string, progressCallback func(float64)) error {
	// 创建 HTTP 请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 错误: %d", resp.StatusCode)
	}

	// 获取内容长度用于进度计算
	contentLength := resp.ContentLength

	// 创建 gzip reader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzReader.Close()

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer outFile.Close()

	// 使用带进度的复制
	if contentLength > 0 && progressCallback != nil {
		written := int64(0)
		buf := make([]byte, 32*1024)
		for {
			n, err := gzReader.Read(buf)
			if n > 0 {
				_, writeErr := outFile.Write(buf[:n])
				if writeErr != nil {
					return writeErr
				}
				written += int64(n)
				// 估算进度（gzip 解压后大小约为压缩前的 3-4 倍）
				progress := float64(written) / float64(contentLength*3) * 100
				if progress > 100 {
					progress = 99
				}
				progressCallback(progress)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	} else {
		if _, err := io.Copy(outFile, gzReader); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
	}

	if progressCallback != nil {
		progressCallback(100)
	}

	return nil
}
