package ffmpeg

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PPTVideoInfo 表示 PPT 中提取的视频信息
type PPTVideoInfo struct {
	Name         string  `json:"name"`
	OriginalPath string  `json:"originalPath"` // 在 zip 中的路径
	TempPath     string  `json:"tempPath"`     // 解压后的临时路径
	Size         int64   `json:"size"`
	Duration     float64 `json:"duration,omitempty"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	Compressed   bool    `json:"compressed"`
	NewSize      int64   `json:"newSize,omitempty"`
	IsMaster     bool    `json:"isMaster"`
}

// PPTCompressor 处理 PPT 视频压缩
type PPTCompressor struct {
	converter    *Converter
	tempDir      string
	progressFunc func(current, total int, currentProgress float64)
	cancelFunc   context.CancelFunc
	currentCmd   *exec.Cmd
}

// NewPPTCompressor 创建新的 PPT 压缩器
func NewPPTCompressor() *PPTCompressor {
	return &PPTCompressor{
		converter: NewConverter(),
	}
}

// SetProgressCallback 设置进度回调
func (p *PPTCompressor) SetProgressCallback(callback func(current, total int, currentProgress float64)) {
	p.progressFunc = callback
}

// ExtractVideosFromPPT 从 PPT 文件中提取视频
func (p *PPTCompressor) ExtractVideosFromPPT(pptPath string) ([]*PPTVideoInfo, string, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "ppt-video-*")
	if err != nil {
		return nil, "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	p.tempDir = tempDir

	// 打开 PPT 文件（实际上是一个 zip 文件）
	zipReader, err := zip.OpenReader(pptPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("打开 PPT 文件失败: %w", err)
	}
	defer zipReader.Close()

	var videos []*PPTVideoInfo

	// 遍历 zip 中的文件
	for _, file := range zipReader.File {
		// 解压所有文件到临时目录
		tempPath := filepath.Join(tempDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(tempPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
			continue
		}

		srcFile, err := file.Open()
		if err != nil {
			continue
		}

		dstFile, err := os.Create(tempPath)
		if err != nil {
			srcFile.Close()
			continue
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()

		if err != nil {
			continue
		}

		// 检查是否在 ppt/media 目录下且是视频文件
		if strings.HasPrefix(file.Name, "ppt/media/") && isVideoFile(file.Name) {
			// 获取文件信息
			info, err := os.Stat(tempPath)
			if err != nil {
				continue
			}

			video := &PPTVideoInfo{
				Name:         filepath.Base(file.Name),
				OriginalPath: file.Name,
				TempPath:     tempPath,
				Size:         info.Size(),
				Compressed:   false,
			}

			// 尝试获取视频信息
			videoInfo, _ := p.converter.GetVideoInfo(tempPath)
			if videoInfo != nil {
				if d, ok := videoInfo["duration"].(string); ok {
					fmt.Sscanf(d, "%f", &video.Duration)
				}
				if w, ok := videoInfo["width"].(float64); ok {
					video.Width = int(w)
				}
				if h, ok := videoInfo["height"].(float64); ok {
					video.Height = int(h)
				}
			}

			videos = append(videos, video)
		}
	}

	if len(videos) == 0 {
		os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("PPT 中未找到视频文件")
	}

	p.identifyMasterVideos(tempDir, videos)

	return videos, tempDir, nil
}

// identifyMasterVideos 识别是否是母版或版式中使用的视频
func (p *PPTCompressor) identifyMasterVideos(tempDir string, videos []*PPTVideoInfo) {
	masterMediaNames := make(map[string]bool)

	relsDirs := []string{
		filepath.Join(tempDir, "ppt", "slideMasters", "_rels"),
		filepath.Join(tempDir, "ppt", "slideLayouts", "_rels"),
	}

	re := regexp.MustCompile(`Target="\.\./media/([^"]+)"`)

	for _, dir := range relsDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".rels") {
				content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err == nil {
					matches := re.FindAllStringSubmatch(string(content), -1)
					for _, match := range matches {
						if len(match) > 1 {
							// URL decode is typically not needed for simple ASCII media names,
							// but spaces might be %20. We will match the raw string directly first.
							// Assuming Microsoft Office uses names like media1.mp4.
							decoded := strings.ReplaceAll(match[1], "%20", " ")
							masterMediaNames[decoded] = true
							masterMediaNames[match[1]] = true
						}
					}
				}
			}
		}
	}

	for _, v := range videos {
		if masterMediaNames[v.Name] {
			v.IsMaster = true
		}
	}
}

// isVideoFile 检查文件是否是视频文件
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mov", ".avi", ".wmv", ".mkv", ".webm", ".m4v", ".mpeg", ".mpg", ".3gp"}
	for _, ve := range videoExts {
		if ext == ve {
			return true
		}
	}
	return false
}

// CompressVideo 压缩单个视频
func (p *PPTCompressor) CompressVideo(video *PPTVideoInfo, quality int) error {
	// 生成输出路径
	outputPath := video.TempPath + ".compressed" + filepath.Ext(video.TempPath)

	// 获取视频信息以支持进度回调
	duration, _ := p.converter.GetVideoDuration(video.TempPath)

	// 构建 FFmpeg 命令参数（用于压缩，不需要透明通道）
	// 使用 H.264 编码，兼容性更好
	args := []string{
		"-i", video.TempPath,
		"-c:v", "libx264",
		"-preset", "medium", // 压缩速度和质量的平衡
		"-crf", fmt.Sprintf("%d", quality), // 质量设置，值越小质量越高
		"-pix_fmt", "yuv420p", // 标准像素格式，兼容性最好
		"-movflags", "+faststart", // 优化网络播放
		"-progress", "pipe:1",
		"-y",
		outputPath,
	}

	cmd := exec.Command(p.converter.ffmpegPath, args...)
	hideWindow(cmd)

	// 捕获 stdout 用于进度
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 FFmpeg 失败: %w", err)
	}

	// 解析进度
	scanner := bufio.NewScanner(stdout)
	timeRegex := regexp.MustCompile(`out_time_ms=(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			if timeMs, err := strconv.ParseFloat(matches[1], 64); err == nil && duration > 0 {
				progress := (timeMs / 1000000) / duration * 100
				if progress > 100 {
					progress = 100
				}
				// 这里不直接调用，由上层控制
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("FFmpeg 压缩失败: %w", err)
	}

	// 获取新文件大小
	info, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("获取压缩后文件信息失败: %w", err)
	}

	// 用压缩后的文件替换原文件
	if err := os.Remove(video.TempPath); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("删除原文件失败: %w", err)
	}

	if err := os.Rename(outputPath, video.TempPath); err != nil {
		return fmt.Errorf("重命名压缩后文件失败: %w", err)
	}

	video.Compressed = true
	video.NewSize = info.Size()

	return nil
}

// CompressVideosWithContext 批量压缩视频（带上下文支持取消）
func (p *PPTCompressor) CompressVideosWithContext(ctx context.Context, videos []*PPTVideoInfo, quality int) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel
	defer func() {
		p.cancelFunc = nil
		p.currentCmd = nil
	}()

	for i, video := range videos {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if p.progressFunc != nil {
			p.progressFunc(i+1, len(videos), 0)
		}

		// 构建 FFmpeg 命令
		outputPath := video.TempPath + ".compressed" + filepath.Ext(video.TempPath)

		args := []string{
			"-i", video.TempPath,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", fmt.Sprintf("%d", quality),
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			"-progress", "pipe:1",
			"-y",
			outputPath,
		}

		cmd := exec.CommandContext(ctx, p.converter.ffmpegPath, args...)
		hideWindow(cmd)
		p.currentCmd = cmd

		// 捕获 stdout 用于进度
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("创建 stdout pipe 失败: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动 FFmpeg 失败: %w", err)
		}

		// 解析进度
		scanner := bufio.NewScanner(stdout)
		timeRegex := regexp.MustCompile(`out_time_ms=(\d+)`)
		duration, _ := p.converter.GetVideoDuration(video.TempPath)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				cmd.Process.Kill()
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
					if p.progressFunc != nil {
						p.progressFunc(i+1, len(videos), progress)
					}
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			os.Remove(outputPath)
			return fmt.Errorf("压缩视频 %s 失败: %w", video.Name, err)
		}

		// 获取新文件大小
		info, err := os.Stat(outputPath)
		if err != nil {
			return fmt.Errorf("获取压缩后文件信息失败: %w", err)
		}

		// 用压缩后的文件替换原文件
		if err := os.Remove(video.TempPath); err != nil {
			os.Remove(outputPath)
			return fmt.Errorf("删除原文件失败: %w", err)
		}

		if err := os.Rename(outputPath, video.TempPath); err != nil {
			return fmt.Errorf("重命名压缩后文件失败: %w", err)
		}

		video.Compressed = true
		video.NewSize = info.Size()

		if p.progressFunc != nil {
			p.progressFunc(i+1, len(videos), 100)
		}
	}
	p.currentCmd = nil

	return nil
}

// Cancel 停止当前的压缩任务
func (p *PPTCompressor) Cancel() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
	if p.currentCmd != nil && p.currentCmd.Process != nil {
		p.currentCmd.Process.Kill()
	}
}

// RepackagePPT 将处理后的文件重新打包为 PPT
func (p *PPTCompressor) RepackagePPT(outputPath string) error {
	if p.tempDir == "" {
		return fmt.Errorf("没有临时目录可打包")
	}

	// 创建输出文件
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历临时目录中的所有文件并添加到 zip
	err = filepath.Walk(p.tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 计算在 zip 中的相对路径
		relPath, err := filepath.Rel(p.tempDir, path)
		if err != nil {
			return err
		}

		// 将路径分隔符转换为正斜杠（zip 标准）
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// 创建 zip 文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate // 压缩存储

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("打包 PPT 失败: %w", err)
	}

	return nil
}

// Cleanup 清理临时文件
func (p *PPTCompressor) Cleanup() {
	if p.tempDir != "" {
		os.RemoveAll(p.tempDir)
		p.tempDir = ""
	}
}

// GetTempDir 获取临时目录路径
func (p *PPTCompressor) GetTempDir() string {
	return p.tempDir
}
