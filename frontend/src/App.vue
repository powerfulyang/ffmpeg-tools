<template>
  <div class="container">
    <header class="header">
      <h1>🎬 MOV to WebM 转换器</h1>
      <p>将 MOV 视频转换为 VP9 编码带透明通道的 WebM 格式</p>
    </header>

    <!-- FFmpeg 初始化状态 -->
    <div class="card" v-if="!ffmpegReady">
      <div class="ffmpeg-status">
        <div v-if="ffmpegError" class="status error">
          <span>❌ FFmpeg 初始化失败: {{ ffmpegError }}</span>
        </div>
        <div v-else class="ffmpeg-loading">
          <span class="loader"></span>
          <div class="ffmpeg-info">
            <p class="ffmpeg-status-text">{{ ffmpegStatus || '正在检查 FFmpeg...' }}</p>
            <div class="progress-bar" v-if="ffmpegProgress > 0">
              <div class="progress-fill" :style="{ width: ffmpegProgress + '%' }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="card" v-else>
      <!-- Initial State: Drop Zone -->
      <div 
        v-if="!inputFile && state === 'idle'"
        class="drop-zone"
        :class="{ 'drag-over': isDragOver }"
        @click="selectFile"
        @dragover.prevent="isDragOver = true"
        @dragleave="isDragOver = false"
        @drop.prevent="handleDrop"
      >
        <span class="drop-zone-icon">📁</span>
        <p class="drop-zone-text">点击选择或拖放 MOV 文件</p>
        <p class="drop-zone-hint">支持带透明通道的 MOV 视频文件</p>
      </div>

      <!-- File Selected State -->
      <div v-else-if="inputFile && state !== 'complete'">
        <div class="file-info">
          <span class="file-icon">🎥</span>
          <div class="file-details">
            <div class="file-name">{{ fileName }}</div>
            <div class="file-meta" v-if="videoInfo">
              {{ videoInfo.width }}×{{ videoInfo.height }} · {{ formatDuration(videoInfo.duration) }}
            </div>
          </div>
          <button class="file-remove" @click="clearFile" title="移除文件">✕</button>
        </div>

        <!-- Video Info Grid -->
        <div class="video-info" v-if="videoInfo">
          <div class="video-info-item">
            <span class="video-info-label">编码格式</span>
            <span class="video-info-value">{{ videoInfo.codec }}</span>
          </div>
          <div class="video-info-item">
            <span class="video-info-label">像素格式</span>
            <span class="video-info-value">{{ videoInfo.pixelFormat }}</span>
          </div>
          <div class="video-info-item">
            <span class="video-info-label">帧率</span>
            <span class="video-info-value">{{ formatFPS(videoInfo.fps) }}</span>
          </div>
          <div class="video-info-item">
            <span class="video-info-label">透明通道</span>
            <span :class="videoInfo.hasAlpha ? 'alpha-badge' : 'no-alpha-badge'">
              {{ videoInfo.hasAlpha ? '✓ 有' : '✗ 无' }}
            </span>
          </div>
        </div>

        <!-- Quality Settings -->
        <div class="settings">
          <div class="quality-warning">
            ⚠️ <strong>注意：数值越低质量越高</strong>，文件也越大
          </div>
          <div class="settings-row">
            <div>
              <span class="settings-label">输出质量 (CRF)</span>
              <div class="settings-hint">推荐范围: 20-30</div>
            </div>
            <div class="range-wrapper">
              <span class="range-label high">高质量</span>
              <div class="range-container">
                <input 
                  type="range" 
                  class="range-slider"
                  v-model.number="quality"
                  :min="0"
                  :max="63"
                />
              </div>
              <span class="range-label low">低质量</span>
            </div>
            <span class="range-value" :class="qualityClass">{{ quality }}</span>
          </div>
        </div>

        <!-- Convert Button -->
        <div class="button-group">
          <button 
            class="btn btn-primary btn-block"
            @click="startConversion"
            :disabled="state === 'converting'"
            v-if="state !== 'converting'"
          >
            <span>🚀</span>
            开始转换
          </button>
          <button 
            class="btn btn-danger btn-block"
            @click="cancelConversion"
            v-else
          >
            <span class="loader"></span>
            取消转换
          </button>
        </div>

        <!-- Progress Bar -->
        <div class="progress-container" v-if="state === 'converting'">
          <div class="progress-header">
            <span class="progress-label">转换进度</span>
            <span class="progress-percent">{{ Math.round(progress) }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progress + '%' }"></div>
          </div>
        </div>

        <!-- Error Message -->
        <div class="status error" v-if="errorMessage" style="margin-top: 16px;">
          ⚠️ {{ errorMessage }}
        </div>
      </div>

      <!-- Complete State -->
      <div v-else-if="state === 'complete'" class="result">
        <span class="result-icon success">✅</span>
        <h2 class="result-title">转换完成!</h2>
        <p class="result-path">{{ outputPath }}</p>
        <div class="result-actions">
          <button class="btn btn-success" @click="openInExplorer">
            📂 打开文件位置
          </button>
          <button class="btn btn-primary" @click="resetAll">
            🔄 继续转换
          </button>
        </div>
      </div>
    </div>

    <!-- Tips -->
    <div class="card" style="padding: 20px;">
      <p style="color: var(--text-secondary); font-size: 0.9rem; text-align: center;">
        💡 提示: 输出文件将保存在源文件同目录下，文件名后缀为 <code>.webm</code>
      </p>
    </div>
  </div>
</template>

<script>
export default {
  name: 'App',
  data() {
    return {
      inputFile: '',
      outputFolder: '',
      outputPath: '',
      quality: 25,
      state: 'idle', // idle, converting, complete
      progress: 0,
      isDragOver: false,
      errorMessage: '',
      videoInfo: null,
      // FFmpeg 状态
      ffmpegReady: false,
      ffmpegStatus: '',
      ffmpegProgress: 0,
      ffmpegError: ''
    }
  },
  computed: {
    fileName() {
      if (!this.inputFile) return ''
      return this.inputFile.split(/[/\\]/).pop()
    },
    qualityClass() {
      if (this.quality <= 20) return 'quality-high'
      if (this.quality <= 35) return 'quality-medium'
      return 'quality-low'
    }
  },
  mounted() {
    // Listen for events from Go backend
    if (window.runtime) {
      // FFmpeg 状态事件
      window.runtime.EventsOn('ffmpeg:progress', (data) => {
        this.ffmpegStatus = data.status
        this.ffmpegProgress = data.progress
      })
      window.runtime.EventsOn('ffmpeg:ready', () => {
        this.ffmpegReady = true
        this.ffmpegStatus = 'FFmpeg 已就绪'
      })
      window.runtime.EventsOn('ffmpeg:error', (error) => {
        this.ffmpegError = error
      })

      // 转换状态事件
      window.runtime.EventsOn('conversion:progress', (progress) => {
        this.progress = progress
      })
      window.runtime.EventsOn('conversion:complete', (path) => {
        this.outputPath = path
        this.state = 'complete'
      })
      window.runtime.EventsOn('conversion:error', (error) => {
        this.errorMessage = error
        this.state = 'idle'
      })
      window.runtime.EventsOn('conversion:cancelled', () => {
        this.errorMessage = '转换已取消'
        this.state = 'idle'
        this.progress = 0
      })
    }

    // 主动检查 FFmpeg 状态，防止错过后端事件
    this.checkFFmpegStatus()
  },
  methods: {
    async checkFFmpegStatus() {
      try {
        const status = await window.go.main.App.CheckFFmpegStatus()
        if (status && status.installed) {
          this.ffmpegReady = true
          this.ffmpegStatus = 'FFmpeg 已就绪'
        }
      } catch (error) {
        console.error('Failed to check FFmpeg status:', error)
      }
    },

    async selectFile() {
      try {
        const file = await window.go.main.App.SelectInputFile()
        if (file) {
          this.inputFile = file
          this.errorMessage = ''
          await this.loadVideoInfo()
        }
      } catch (error) {
        this.errorMessage = '选择文件失败: ' + error.message
      }
    },
    
    handleDrop(event) {
      this.isDragOver = false
      // Note: Wails handles file drops differently, using the select dialog
      this.selectFile()
    },
    
    async loadVideoInfo() {
      try {
        const info = await window.go.main.App.GetVideoInfo(this.inputFile)
        this.videoInfo = info
      } catch (error) {
        console.error('Failed to load video info:', error)
      }
    },
    
    clearFile() {
      this.inputFile = ''
      this.videoInfo = null
      this.errorMessage = ''
      this.state = 'idle'
      this.progress = 0
    },
    
    async startConversion() {
      if (!this.inputFile) {
        this.errorMessage = '请先选择一个文件'
        return
      }
      
      this.state = 'converting'
      this.progress = 0
      this.errorMessage = ''
      
      try {
        const result = await window.go.main.App.ConvertToWebM(
          this.inputFile, 
          this.outputFolder, 
          this.quality
        )
        
        if (result.success) {
          this.outputPath = result.outputPath
          this.state = 'complete'
        } else {
          this.errorMessage = result.message
          this.state = 'idle'
        }
      } catch (error) {
        this.errorMessage = '转换失败: ' + error.message
        this.state = 'idle'
      }
    },
    
    async cancelConversion() {
      try {
        await window.go.main.App.CancelConversion()
      } catch (error) {
        console.error('Failed to cancel conversion:', error)
      }
    },
    
    async openInExplorer() {
      try {
        await window.go.main.App.OpenFileInExplorer(this.outputPath)
      } catch (error) {
        console.error('Failed to open explorer:', error)
      }
    },
    
    resetAll() {
      this.inputFile = ''
      this.outputPath = ''
      this.videoInfo = null
      this.state = 'idle'
      this.progress = 0
      this.errorMessage = ''
    },
    
    formatDuration(seconds) {
      if (!seconds) return '--'
      const secs = parseFloat(seconds)
      const mins = Math.floor(secs / 60)
      const remainingSecs = Math.floor(secs % 60)
      return `${mins}:${remainingSecs.toString().padStart(2, '0')}`
    },
    
    formatFPS(fps) {
      if (!fps) return '--'
      // FPS is usually in format "30/1" or "30000/1001"
      const parts = fps.split('/')
      if (parts.length === 2) {
        const result = parseFloat(parts[0]) / parseFloat(parts[1])
        return result.toFixed(2) + ' fps'
      }
      return fps + ' fps'
    }
  }
}
</script>
