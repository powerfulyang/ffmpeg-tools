# ES Tools - MOV to WebM 转换器

一个使用 Wails 构建的桌面应用，用于将 MOV 视频转换成 VP9 编码带透明通道的 WebM 文件。

## 功能特性

- 🎬 将 MOV 视频转换为 VP9 编码的 WebM 格式
- 🔮 保留透明通道（Alpha Channel）
- 📊 实时显示转换进度
- 🎛️ 可调节输出质量（CRF 值）
- 📁 自动检测视频信息（分辨率、编码、帧率等）

## 技术栈

- **后端**: Go + Wails v2
- **前端**: Vue 3 + Vite
- **视频处理**: FFmpeg

## 项目结构

```
es-tools/
├── main.go                     # Wails 应用入口
├── app.go                      # 应用逻辑层
├── wails.json                  # Wails 配置
├── go.mod                      # Go 模块
├── internal/
│   └── ffmpeg/
│       └── converter.go        # FFmpeg 封装
├── frontend/
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   └── src/
│       ├── main.js
│       ├── style.css
│       └── App.vue
└── resources/
    └── ffmpeg/                 # FFmpeg 二进制文件
        ├── ffmpeg.exe
        └── ffprobe.exe
```

## 开发环境设置

### 前提条件

- Go 1.21+
- Node.js 18+
- Wails CLI v2

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 添加 FFmpeg

运行 Go 下载脚本自动下载各平台的 FFmpeg：

```bash
go run scripts/download_ffmpeg.go
```

FFmpeg 二进制文件将保存到对应平台目录：
- `resources/ffmpeg/windows-amd64/`
- `resources/ffmpeg/darwin-amd64/`
- `resources/ffmpeg/darwin-arm64/`

### 开发模式

```bash
# 方式1: 使用 Wails CLI（推荐，如果已安装）
wails dev

# 方式2: 手动运行
cd frontend && pnpm run dev  # 终端1
go build -tags dev -gcflags "all=-N -l" && ./es-tools.exe  # 终端2
```

### 构建生产版本

```bash
# 方式1: 使用 Wails CLI
wails build

# 方式2: 手动构建
cd frontend && pnpm run build
go build -tags desktop,production -ldflags "-w -s -H windowsgui" -o build/bin/es-tools.exe .

# 方式3: 使用构建脚本
.\scripts\build.ps1
```

生成的可执行文件位于 `build/bin/` 目录。

## FFmpeg 转换参数说明

转换使用以下 FFmpeg 参数：

```bash
ffmpeg -i input.mov \
  -c:v libvpx-vp9 \      # 使用 VP9 编码
  -pix_fmt yuva420p \    # 支持透明通道的像素格式
  -crf <quality> \       # 质量参数 (0-63，越低越好)
  -b:v 0 \               # 使用 CRF 模式
  -auto-alt-ref 0 \      # Alpha 通道所需
  -an \                  # 无音频
  output.webm
```

## 许可证

MIT License
