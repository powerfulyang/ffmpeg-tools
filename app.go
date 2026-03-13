package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"es-tools/internal/ffmpeg"

	"os/exec"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	converter     *ffmpeg.Converter
	pptCompressor *ffmpeg.PPTCompressor
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.converter = ffmpeg.NewConverter()

	// 在后台检查并下载 FFmpeg
	go a.ensureFFmpeg()
}

// ensureFFmpeg 确保 FFmpeg 已安装
func (a *App) ensureFFmpeg() {
	// 发送初始化状态
	runtime.EventsEmit(a.ctx, "ffmpeg:status", "checking")

	err := ffmpeg.EnsureFFmpeg(func(status string, progress float64) {
		runtime.EventsEmit(a.ctx, "ffmpeg:progress", map[string]interface{}{
			"status":   status,
			"progress": progress,
		})
	})

	if err != nil {
		runtime.EventsEmit(a.ctx, "ffmpeg:error", err.Error())
	} else {
		// 重新初始化 converter 以使用新下载的 FFmpeg
		a.converter = ffmpeg.NewConverter()
		runtime.EventsEmit(a.ctx, "ffmpeg:ready", true)
	}
}

// CheckFFmpegStatus 检查 FFmpeg 状态
func (a *App) CheckFFmpegStatus() map[string]interface{} {
	installed := a.converter.CheckFFmpegInstalled()
	return map[string]interface{}{
		"installed": installed,
		"path":      a.converter.GetFFmpegPath(),
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Cancel any running ffmpeg processes to prevent orphaned processes when exiting the program
	if a.converter != nil {
		a.converter.Cancel()
	}
	if a.pptCompressor != nil {
		a.pptCompressor.Cancel()
	}
}

// SelectInputFile opens a file dialog to select MOV file
func (a *App) SelectInputFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 MOV 文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "MOV 视频文件",
				Pattern:     "*.mov",
			},
			{
				DisplayName: "所有视频文件",
				Pattern:     "*.mov;*.mp4;*.avi;*.mkv",
			},
		},
	})
	return file, err
}

// SelectInputFiles opens a file dialog to select multiple MOV files
func (a *App) SelectInputFiles() ([]string, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 MOV 文件（可多选）",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "MOV 视频文件",
				Pattern:     "*.mov",
			},
			{
				DisplayName: "所有视频文件",
				Pattern:     "*.mov;*.mp4;*.avi;*.mkv",
			},
		},
	})
	return files, err
}

// SelectOutputFolder opens a folder dialog to select output directory
func (a *App) SelectOutputFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择输出文件夹",
	})
	return folder, err
}

// ConvertResult represents the result of a conversion
type ConvertResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	OutputPath string `json:"outputPath"`
}

// ConvertToWebM converts a MOV file to VP9 WebM with alpha channel
func (a *App) ConvertToWebM(inputPath string, outputFolder string, quality int) ConvertResult {
	if inputPath == "" {
		return ConvertResult{Success: false, Message: "请选择输入文件"}
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return ConvertResult{Success: false, Message: "输入文件不存在"}
	}

	// Generate output path
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputFolder, baseName+".webm")

	// If output folder is empty, use the same folder as input
	if outputFolder == "" {
		outputPath = filepath.Join(filepath.Dir(inputPath), baseName+".webm")
	}

	// Emit progress event
	runtime.EventsEmit(a.ctx, "conversion:start", inputPath)

	// Convert the file
	err := a.converter.ConvertMOVToVP9WebM(inputPath, outputPath, quality, func(progress float64) {
		runtime.EventsEmit(a.ctx, "conversion:progress", progress)
	})

	if err != nil {
		runtime.EventsEmit(a.ctx, "conversion:error", err.Error())
		return ConvertResult{
			Success: false,
			Message: fmt.Sprintf("转换失败: %v", err),
		}
	}

	runtime.EventsEmit(a.ctx, "conversion:complete", outputPath)
	return ConvertResult{
		Success:    true,
		Message:    "转换完成",
		OutputPath: outputPath,
	}
}

// GetVideoInfo returns information about a video file
func (a *App) GetVideoInfo(inputPath string) (map[string]interface{}, error) {
	return a.converter.GetVideoInfo(inputPath)
}

// OpenFileInExplorer opens the file location in Windows Explorer
func (a *App) OpenFileInExplorer(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		// 使用 /select, 可以选中该文件
		cmd = exec.Command("explorer", "/select,", absPath)
	case "darwin":
		cmd = exec.Command("open", "-R", absPath)
	default: // Linux
		cmd = exec.Command("xdg-open", filepath.Dir(absPath))
	}
	cmd.Run()
}

// CancelConversion cancels the current conversion task
func (a *App) CancelConversion() {
	if a.converter != nil {
		a.converter.Cancel()
		runtime.EventsEmit(a.ctx, "conversion:cancelled", true)
	}
}

// ========== PPT 视频压缩功能 ==========

// PPTCompressResult 表示 PPT 压缩结果
type PPTCompressResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	OutputPath string                 `json:"outputPath"`
	Videos     []*ffmpeg.PPTVideoInfo `json:"videos,omitempty"`
	TotalSaved int64                  `json:"totalSaved,omitempty"`
}

// SelectPPTFiles 选择多个 PPT 文件
func (a *App) SelectPPTFiles() ([]string, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 PPT 文件（可多选）",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PPT 文件",
				Pattern:     "*.pptx;*.ppt",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	return files, err
}

// ProcessPPTFile 一键处理单个 PPT：提取、压缩、重新打包
func (a *App) ProcessPPTFile(pptPath string, quality int, onlyMaster bool) PPTCompressResult {
	if pptPath == "" {
		return PPTCompressResult{Success: false, Message: "请选择 PPT 文件"}
	}

	if _, err := os.Stat(pptPath); os.IsNotExist(err) {
		return PPTCompressResult{Success: false, Message: "PPT 文件不存在"}
	}

	compressor := ffmpeg.NewPPTCompressor()
	a.pptCompressor = compressor

	runtime.EventsEmit(a.ctx, "ppt:status", "extracting")

	// 1. 提取视频并识别母版
	videos, _, err := compressor.ExtractVideosFromPPT(pptPath)
	if err != nil {
		a.pptCompressor.Cleanup()
		a.pptCompressor = nil
		return PPTCompressResult{Success: false, Message: fmt.Sprintf("提取视频失败: %v", err)}
	}

	// 2. 筛选需要压缩的视频
	var videosToCompress []*ffmpeg.PPTVideoInfo
	for _, v := range videos {
		if onlyMaster && !v.IsMaster {
			continue // 跳过非母版视频
		}
		videosToCompress = append(videosToCompress, v)
	}

	if len(videosToCompress) == 0 {
		a.pptCompressor.Cleanup()
		a.pptCompressor = nil
		return PPTCompressResult{
			Success:    true,
			Message:    "没有符合条件的视频需要压缩",
			OutputPath: pptPath, // 未改变
			Videos:     videos,
			TotalSaved: 0,
		}
	}

	// 3. 压缩视频
	compressor.SetProgressCallback(func(current, total int, currentProgress float64) {
		runtime.EventsEmit(a.ctx, "ppt:progress", map[string]interface{}{
			"current":         current,
			"total":           total,
			"currentProgress": currentProgress,
			"overallProgress": (float64(current-1) + currentProgress/100) / float64(total) * 100,
		})
	})

	runtime.EventsEmit(a.ctx, "ppt:status", "compressing")

	ctx := context.Background()
	if err := compressor.CompressVideosWithContext(ctx, videosToCompress, quality); err != nil {
		a.pptCompressor.Cleanup()
		a.pptCompressor = nil
		if strings.Contains(err.Error(), "取消") {
			return PPTCompressResult{Success: false, Message: "取消压缩"}
		}
		return PPTCompressResult{Success: false, Message: fmt.Sprintf("压缩视频失败: %v", err)}
	}

	// 4. 打包文件
	runtime.EventsEmit(a.ctx, "ppt:status", "repackaging")

	baseName := strings.TrimSuffix(filepath.Base(pptPath), filepath.Ext(pptPath))
	dir := filepath.Dir(pptPath)
	outputPath := filepath.Join(dir, baseName+"_compressed.pptx")

	if err := compressor.RepackagePPT(outputPath); err != nil {
		a.pptCompressor.Cleanup()
		a.pptCompressor = nil
		return PPTCompressResult{Success: false, Message: fmt.Sprintf("重新打包 PPT 失败: %v", err)}
	}

	// 计算节省
	var totalSaved int64
	for _, v := range videosToCompress {
		if v.Compressed {
			totalSaved += v.Size - v.NewSize
		}
	}

	a.pptCompressor.Cleanup()
	a.pptCompressor = nil

	runtime.EventsEmit(a.ctx, "ppt:complete", outputPath)

	return PPTCompressResult{
		Success:    true,
		Message:    fmt.Sprintf("成功压缩 %d 个视频", len(videosToCompress)),
		OutputPath: outputPath,
		Videos:     videosToCompress,
		TotalSaved: totalSaved,
	}
}

// CancelPPTCompression 取消 PPT 压缩
func (a *App) CancelPPTCompression() {
	if a.pptCompressor != nil {
		a.pptCompressor.Cancel()
		runtime.EventsEmit(a.ctx, "ppt:cancelled", true)
	}
}
