# FFmpeg Tools - Project Documentation

## 🚀 Overview
A desktop utility built with **Wails**, **Vue 3**, and **FFmpeg** for specialized video processing tasks.

## ✨ Key Features
- **MOV to WebM Conversion**: High-quality conversion with alpha transparency support (VP9).
- **Batch PPT Video Compression**: 
  - Supports processing multiple `.pptx` files.
  - **Master Only Mode**: Intelligent filtering to only compress videos found in Slide Masters or Layouts.
  - **Automatic Output**: Saves compressed files next to originals with `_compressed` suffix.
- **Robust Cancellation**: Active `ffmpeg` processes are immediately killed upon user cancellation to free system resources.

## 🛠️ Technical Stack
- **Backend**: Go (Wails)
- **Frontend**: Vue 3, Vite
- **Styling**: UnoCSS (Atomic CSS)
- **Processing**: FFmpeg (Self-managed/Downloader included)

## 📂 Project Structure
- `/app.go`: Main backend entry point and Wails binding methods.
- `/internal/ffmpeg`: Core video processing logic.
  - `converter.go`: MOV conversion and FFmpeg management.
  - `ppt_compressor.go`: PPT extraction, master-video identification, and repackaging.
- `/frontend`: Vue application.
  - `src/App.vue`: Main UI orchestrator.
  - `uno.config.ts`: UnoCSS configuration.

## ⚙️ Development Highlights
- **Process Management**: Uses `hideWindow` on Windows to run FFmpeg silently.
- **Context Cancellation**: Full support for `context.Context` propagation to terminate background tasks.
- **Master Video Detection**: Scans relationships (`ppt/slideMasters/_rels/*.rels`) to determine media importance.

---
*Updated: 2026-03-13*
