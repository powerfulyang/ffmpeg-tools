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
        v-if="files.length === 0 && state === 'idle'"
        class="drop-zone"
        :class="{ 'drag-over': isDragOver }"
        @click="selectFiles"
        @dragover.prevent="isDragOver = true"
        @dragleave="isDragOver = false"
        @drop.prevent="handleDrop"
      >
        <span class="drop-zone-icon">📁</span>
        <p class="drop-zone-text">点击选择或拖放 MOV 文件</p>
        <p class="drop-zone-hint">支持选择多个带透明通道的 MOV 视频文件</p>
      </div>

      <!-- File List State -->
      <div v-else-if="files.length > 0 && state !== 'complete'">
        <!-- File List Header -->
        <div class="file-list-header">
          <span class="file-count">
            <span class="file-count-icon">📂</span>
            已选择 {{ files.length }} 个文件
          </span>
          <div class="file-list-actions">
            <button class="btn-text" @click="selectFiles" :disabled="state === 'converting'">
              <span>➕</span> 添加
            </button>
            <button class="btn-text danger" @click="clearAllFiles" :disabled="state === 'converting'">
              <span>🗑️</span> 清空
            </button>
          </div>
        </div>

        <!-- File List -->
        <div class="file-list">
          <div 
            class="file-item" 
            v-for="(file, index) in files" 
            :key="file.path"
            :class="{ 
              'converting': currentFileIndex === index && state === 'converting',
              'completed': file.status === 'completed',
              'error': file.status === 'error'
            }"
          >
            <div class="file-item-content">
              <span class="file-status-icon">
                <template v-if="file.status === 'completed'">✅</template>
                <template v-else-if="file.status === 'error'">❌</template>
                <template v-else-if="currentFileIndex === index && state === 'converting'">
                  <span class="file-loader"></span>
                </template>
                <template v-else>🎥</template>
              </span>
              <div class="file-item-details">
                <div class="file-item-name">{{ getFileName(file.path) }}</div>
                <div class="file-item-meta" v-if="file.info">
                  {{ file.info.width }}×{{ file.info.height }} · {{ formatDuration(file.info.duration) }}
                  <span :class="file.info.hasAlpha ? 'alpha-badge-small' : 'no-alpha-badge-small'">
                    {{ file.info.hasAlpha ? '透明' : '不透明' }}
                  </span>
                </div>
                <div class="file-item-error" v-if="file.errorMessage">
                  {{ file.errorMessage }}
                </div>
                <!-- Per-file progress bar -->
                <div class="file-item-progress" v-if="currentFileIndex === index && state === 'converting'">
                  <div class="progress-bar small">
                    <div class="progress-fill" :style="{ width: currentProgress + '%' }"></div>
                  </div>
                  <span class="progress-text">{{ Math.round(currentProgress) }}%</span>
                </div>
              </div>
              <button 
                class="file-item-remove" 
                @click="removeFile(index)" 
                title="移除文件"
                :disabled="state === 'converting'"
              >✕</button>
            </div>
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
                  :disabled="state === 'converting'"
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
            @click="startBatchConversion"
            :disabled="state === 'converting' || files.length === 0"
            v-if="state !== 'converting'"
          >
            <span>🚀</span>
            开始批量转换 ({{ files.length }} 个文件)
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

        <!-- Overall Progress -->
        <div class="progress-container" v-if="state === 'converting'">
          <div class="progress-header">
            <span class="progress-label">总体进度</span>
            <span class="progress-percent">{{ completedCount }} / {{ files.length }}</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill overall" :style="{ width: overallProgress + '%' }"></div>
          </div>
        </div>

        <!-- Global Error Message -->
        <div class="status error" v-if="errorMessage" style="margin-top: 16px;">
          ⚠️ {{ errorMessage }}
        </div>
      </div>

      <!-- Complete State -->
      <div v-else-if="state === 'complete'" class="result">
        <span class="result-icon success">✅</span>
        <h2 class="result-title">批量转换完成!</h2>
        <div class="result-summary">
          <p class="result-stats">
            成功: <span class="success-count">{{ successCount }}</span> · 
            失败: <span class="error-count">{{ errorCount }}</span>
          </p>
        </div>
        <div class="result-actions">
          <button class="btn btn-success" @click="openOutputFolder" v-if="successCount > 0">
            📂 打开输出目录
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
      files: [], // { path, info, status, outputPath, errorMessage }
      outputFolder: '',
      quality: 25,
      state: 'idle', // idle, converting, complete
      currentProgress: 0,
      currentFileIndex: -1,
      isDragOver: false,
      errorMessage: '',
      // FFmpeg 状态
      ffmpegReady: false,
      ffmpegStatus: '',
      ffmpegProgress: 0,
      ffmpegError: ''
    }
  },
  computed: {
    qualityClass() {
      if (this.quality <= 20) return 'quality-high'
      if (this.quality <= 35) return 'quality-medium'
      return 'quality-low'
    },
    completedCount() {
      return this.files.filter(f => f.status === 'completed' || f.status === 'error').length
    },
    successCount() {
      return this.files.filter(f => f.status === 'completed').length
    },
    errorCount() {
      return this.files.filter(f => f.status === 'error').length
    },
    overallProgress() {
      if (this.files.length === 0) return 0
      const baseProgress = (this.completedCount / this.files.length) * 100
      const currentFileProgress = (this.currentProgress / 100) / this.files.length * 100
      return Math.min(baseProgress + currentFileProgress, 100)
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
        this.currentProgress = progress
      })
      window.runtime.EventsOn('conversion:complete', (path) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'completed'
          this.files[this.currentFileIndex].outputPath = path
        }
      })
      window.runtime.EventsOn('conversion:error', (error) => {
        if (this.currentFileIndex >= 0 && this.currentFileIndex < this.files.length) {
          this.files[this.currentFileIndex].status = 'error'
          this.files[this.currentFileIndex].errorMessage = error
        }
      })
      window.runtime.EventsOn('conversion:cancelled', () => {
        this.errorMessage = '转换已取消'
        this.state = 'idle'
        this.currentProgress = 0
        this.currentFileIndex = -1
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

    async selectFiles() {
      try {
        const selectedFiles = await window.go.main.App.SelectInputFiles()
        if (selectedFiles && selectedFiles.length > 0) {
          this.errorMessage = ''
          // 添加新文件，避免重复
          for (const filePath of selectedFiles) {
            if (!this.files.find(f => f.path === filePath)) {
              const fileObj = {
                path: filePath,
                info: null,
                status: 'pending',
                outputPath: '',
                errorMessage: ''
              }
              this.files.push(fileObj)
              // 异步加载视频信息
              this.loadVideoInfo(fileObj)
            }
          }
        }
      } catch (error) {
        this.errorMessage = '选择文件失败: ' + error.message
      }
    },
    
    handleDrop(event) {
      this.isDragOver = false
      // Note: Wails handles file drops differently, using the select dialog
      this.selectFiles()
    },
    
    async loadVideoInfo(fileObj) {
      try {
        const info = await window.go.main.App.GetVideoInfo(fileObj.path)
        fileObj.info = info
      } catch (error) {
        console.error('Failed to load video info:', error)
      }
    },
    
    getFileName(filePath) {
      if (!filePath) return ''
      return filePath.split(/[/\\]/).pop()
    },

    removeFile(index) {
      if (this.state === 'converting') return
      this.files.splice(index, 1)
    },

    clearAllFiles() {
      if (this.state === 'converting') return
      this.files = []
      this.errorMessage = ''
    },
    
    async startBatchConversion() {
      if (this.files.length === 0) {
        this.errorMessage = '请先选择文件'
        return
      }
      
      this.state = 'converting'
      this.currentProgress = 0
      this.currentFileIndex = 0
      this.errorMessage = ''
      
      // 重置所有文件状态
      for (const file of this.files) {
        file.status = 'pending'
        file.errorMessage = ''
      }
      
      // 依次转换每个文件
      for (let i = 0; i < this.files.length; i++) {
        if (this.state !== 'converting') break // 被取消
        
        this.currentFileIndex = i
        this.currentProgress = 0
        
        try {
          const result = await window.go.main.App.ConvertToWebM(
            this.files[i].path, 
            this.outputFolder, 
            this.quality
          )
          
          if (result.success) {
            this.files[i].status = 'completed'
            this.files[i].outputPath = result.outputPath
          } else {
            this.files[i].status = 'error'
            this.files[i].errorMessage = result.message
          }
        } catch (error) {
          this.files[i].status = 'error'
          this.files[i].errorMessage = '转换失败: ' + error.message
        }
      }
      
      if (this.state === 'converting') {
        this.state = 'complete'
        this.currentFileIndex = -1
      }
    },
    
    async cancelConversion() {
      try {
        await window.go.main.App.CancelConversion()
        this.state = 'idle'
        this.currentFileIndex = -1
        this.currentProgress = 0
      } catch (error) {
        console.error('Failed to cancel conversion:', error)
      }
    },
    
    async openOutputFolder() {
      // 打开第一个成功转换的文件所在目录
      const successFile = this.files.find(f => f.status === 'completed')
      if (successFile && successFile.outputPath) {
        try {
          await window.go.main.App.OpenFileInExplorer(successFile.outputPath)
        } catch (error) {
          console.error('Failed to open explorer:', error)
        }
      }
    },
    
    resetAll() {
      this.files = []
      this.state = 'idle'
      this.currentProgress = 0
      this.currentFileIndex = -1
      this.errorMessage = ''
    },
    
    formatDuration(seconds) {
      if (!seconds) return '--'
      const secs = parseFloat(seconds)
      const mins = Math.floor(secs / 60)
      const remainingSecs = Math.floor(secs % 60)
      return `${mins}:${remainingSecs.toString().padStart(2, '0')}`
    }
  }
}
</script>
